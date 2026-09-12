package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/haidodev/authz-service/internal/authz"
)

type fakeEngine struct {
	allowed        bool
	roleExists     bool
	err            error
	assigned       bool
	revoked        bool
	roleCreated    bool
	roleAssigned   bool
	rolePermission authz.Permission
}

func (f *fakeEngine) Initialize(context.Context) error { return nil }
func (f *fakeEngine) Check(context.Context, uuid.UUID, uuid.UUID, authz.Permission) (bool, error) {
	return f.allowed, f.err
}
func (f *fakeEngine) AssignAdmin(context.Context, uuid.UUID, uuid.UUID) error {
	f.assigned = true
	return f.err
}
func (f *fakeEngine) RevokeAdmin(context.Context, uuid.UUID, uuid.UUID) error {
	f.revoked = true
	return f.err
}
func (f *fakeEngine) CreateRole(context.Context, uuid.UUID, uuid.UUID) error {
	f.roleCreated = true
	return f.err
}
func (f *fakeEngine) RoleExists(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return f.roleExists, f.err
}
func (f *fakeEngine) AssignRole(context.Context, uuid.UUID, uuid.UUID) error {
	f.roleAssigned = true
	return f.err
}
func (f *fakeEngine) RevokeRole(context.Context, uuid.UUID, uuid.UUID) error { return f.err }
func (f *fakeEngine) GrantRolePermission(_ context.Context, _ uuid.UUID, _ uuid.UUID, permission authz.Permission) error {
	f.rolePermission = permission
	return f.err
}
func (f *fakeEngine) RevokeRolePermission(context.Context, uuid.UUID, uuid.UUID, authz.Permission) error {
	return f.err
}

const (
	testActor = "11111111-1111-1111-1111-111111111111"
	testOrg   = "22222222-2222-2222-2222-222222222222"
	testUser  = "33333333-3333-3333-3333-333333333333"
)

func testRouter(engine *fakeEngine) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	org := uuid.MustParse(testOrg)
	user := uuid.MustParse(testActor)
	NewHandler(engine, org, user).Register(router)
	return router
}

func TestCheckReturnsDecision(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/v1/check", strings.NewReader(`{"subjectId":"`+testActor+`","organizationId":"`+testOrg+`","permission":"list_users"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	testRouter(&fakeEngine{allowed: true}).ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"allowed":true`) {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
}

func TestAssignAdminRequiresExistingAdmin(t *testing.T) {
	engine := &fakeEngine{allowed: false}
	request := httptest.NewRequest(http.MethodPut, "/v1/organizations/"+testOrg+"/users/"+testUser+"/roles/admin", nil)
	request.Header.Set("X-User-ID", testActor)
	response := httptest.NewRecorder()
	testRouter(engine).ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
	}
	if engine.assigned {
		t.Fatal("role was assigned without admin permission")
	}
}

func TestBootstrapAdminRequiresInternalToken(t *testing.T) {
	engine := &fakeEngine{}
	router := gin.New()
	NewHandler(engine, uuid.MustParse(testOrg), uuid.MustParse(testActor), "identity-token").Register(router)
	request := httptest.NewRequest(http.MethodPost, "/v1/internal/organizations/"+testOrg+"/bootstrap-admin/"+testUser, nil)
	request.Header.Set("X-Internal-Token", "identity-token")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent || !engine.assigned {
		t.Fatalf("response = %d, assigned = %v", response.Code, engine.assigned)
	}
}

func TestBootstrapAdminCannotBeRevoked(t *testing.T) {
	engine := &fakeEngine{allowed: true}
	request := httptest.NewRequest(http.MethodDelete, "/v1/organizations/"+testOrg+"/users/"+testActor+"/roles/admin", nil)
	request.Header.Set("X-User-ID", testActor)
	response := httptest.NewRecorder()
	testRouter(engine).ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if engine.revoked {
		t.Fatal("bootstrap admin was revoked")
	}
}

func TestCheckMapsEngineFailure(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/v1/check", strings.NewReader(`{"subjectId":"`+testActor+`","organizationId":"`+testOrg+`","permission":"list_users"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	testRouter(&fakeEngine{err: errors.New("OpenFGA offline")}).ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
}

func TestCreateCustomRoleRequiresAdmin(t *testing.T) {
	engine := &fakeEngine{allowed: false}
	request := httptest.NewRequest(http.MethodPost, "/v1/organizations/"+testOrg+"/roles", nil)
	request.Header.Set("X-User-ID", testActor)
	response := httptest.NewRecorder()
	testRouter(engine).ServeHTTP(response, request)
	if response.Code != http.StatusForbidden || engine.roleCreated {
		t.Fatalf("status = %d, roleCreated = %v", response.Code, engine.roleCreated)
	}
}

func TestCreateCustomRole(t *testing.T) {
	engine := &fakeEngine{allowed: true}
	request := httptest.NewRequest(http.MethodPost, "/v1/organizations/"+testOrg+"/roles", nil)
	request.Header.Set("X-User-ID", testActor)
	response := httptest.NewRecorder()
	testRouter(engine).ServeHTTP(response, request)
	if response.Code != http.StatusCreated || !engine.roleCreated {
		t.Fatalf("status = %d, roleCreated = %v", response.Code, engine.roleCreated)
	}
	if !strings.Contains(response.Body.String(), `"id"`) {
		t.Fatalf("response = %s, want generated role ID", response.Body.String())
	}
}

func TestAssignCustomRoleRequiresExistingOrganizationRole(t *testing.T) {
	engine := &fakeEngine{allowed: true, roleExists: false}
	roleID := "44444444-4444-4444-4444-444444444444"
	request := httptest.NewRequest(http.MethodPut, "/v1/organizations/"+testOrg+"/roles/"+roleID+"/users/"+testUser, nil)
	request.Header.Set("X-User-ID", testActor)
	response := httptest.NewRecorder()
	testRouter(engine).ServeHTTP(response, request)
	if response.Code != http.StatusNotFound || engine.roleAssigned {
		t.Fatalf("status = %d, roleAssigned = %v", response.Code, engine.roleAssigned)
	}
}

func TestGrantCustomRolePermission(t *testing.T) {
	engine := &fakeEngine{allowed: true, roleExists: true}
	roleID := "44444444-4444-4444-4444-444444444444"
	request := httptest.NewRequest(http.MethodPut, "/v1/organizations/"+testOrg+"/roles/"+roleID+"/permissions/create_user", nil)
	request.Header.Set("X-User-ID", testActor)
	response := httptest.NewRecorder()
	testRouter(engine).ServeHTTP(response, request)
	if response.Code != http.StatusNoContent || engine.rolePermission != authz.PermissionCreateUsers {
		t.Fatalf("status = %d, rolePermission = %q", response.Code, engine.rolePermission)
	}
}
