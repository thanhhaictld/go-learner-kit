package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/haidodev/user-service/internal/authz"
	"github.com/haidodev/user-service/internal/domain"
	"github.com/haidodev/user-service/internal/repository"
	"github.com/haidodev/user-service/internal/service"
)

type fakeUserRepository struct {
	created   *domain.User
	createErr error
}

func (repo *fakeUserRepository) CreateUser(_ context.Context, user *domain.User) error {
	repo.created = user
	return repo.createErr
}

func (repo *fakeUserRepository) GetUserByID(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*domain.User, error) {
	return nil, nil
}

func (repo *fakeUserRepository) ListUsers(_ context.Context, _ uuid.UUID) ([]domain.User, error) {
	return nil, nil
}

type fakeAuthorizer struct {
	err    error
	checks []authz.Permission
}

func (authorizer *fakeAuthorizer) Check(_ context.Context, _, _ uuid.UUID, permission authz.Permission) error {
	authorizer.checks = append(authorizer.checks, permission)
	return authorizer.err
}

type errorResponse struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Errors  map[string]string `json:"errors"`
}

func newTestRouter(repo *fakeUserRepository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterRoutes(router, NewHandler(service.NewUserService(repo), &fakeAuthorizer{}))
	return router
}

func TestCreateUserValidation(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantErrors map[string]string
	}{
		{
			name:       "missing email",
			body:       `{"name":"Ada"}`,
			wantErrors: map[string]string{"email": "is required"},
		},
		{
			name:       "invalid email",
			body:       `{"email":"not-an-email","name":"Ada"}`,
			wantErrors: map[string]string{"email": "must be a valid email address"},
		},
		{
			name:       "blank name",
			body:       `{"email":"ada@example.com","name":" \t "}`,
			wantErrors: map[string]string{"name": "must not be blank"},
		},
		{
			name:       "multiple invalid fields",
			body:       `{}`,
			wantErrors: map[string]string{"email": "is required", "name": "must not be blank"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeUserRepository{}
			response := performCreateUserRequest(newTestRouter(repo), tt.body)

			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
			}
			if repo.created != nil {
				t.Fatal("repository received a user for an invalid request")
			}

			var body errorResponse
			decodeResponse(t, response, &body)
			if body.Code != "invalid_request" || body.Message != "validation failed" {
				t.Fatalf("error = %#v, want invalid_request validation failed", body)
			}
			if !mapsEqual(body.Errors, tt.wantErrors) {
				t.Fatalf("errors = %#v, want %#v", body.Errors, tt.wantErrors)
			}
		})
	}
}

func TestCreateUserTrimsName(t *testing.T) {
	repo := &fakeUserRepository{}
	response := performCreateUserRequest(newTestRouter(repo), `{"email":"ada@example.com","name":"  Ada  "}`)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusCreated)
	}
	if repo.created == nil {
		t.Fatal("repository did not receive a user")
	}
	if repo.created.Name != "Ada" {
		t.Errorf("created name = %q, want %q", repo.created.Name, "Ada")
	}
}

func TestCreateUserMalformedJSON(t *testing.T) {
	repo := &fakeUserRepository{}
	response := performCreateUserRequest(newTestRouter(repo), `{"email":`)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if repo.created != nil {
		t.Fatal("repository received a user for malformed JSON")
	}

	var body errorResponse
	decodeResponse(t, response, &body)
	if body.Code != "invalid_request" || body.Errors != nil {
		t.Fatalf("error = %#v, want invalid_request without field errors", body)
	}
}

func TestCreateUserDuplicateEmail(t *testing.T) {
	repo := &fakeUserRepository{createErr: repository.ErrEmailAlreadyExists}
	response := performCreateUserRequest(newTestRouter(repo), `{"email":"ada@example.com","name":"Ada"}`)

	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusConflict)
	}

	var body errorResponse
	decodeResponse(t, response, &body)
	if body.Code != "conflict" || body.Message != "email already exists" {
		t.Fatalf("error = %#v, want conflict response", body)
	}
	if !mapsEqual(body.Errors, map[string]string{"email": "already exists"}) {
		t.Fatalf("errors = %#v, want duplicate email error", body.Errors)
	}
}

func performCreateUserRequest(router http.Handler, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-User-ID", "11111111-1111-1111-1111-111111111111")
	request.Header.Set("X-Organization-ID", "22222222-2222-2222-2222-222222222222")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func TestCreateUserAuthorizationFailureDoesNotCreateUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &fakeUserRepository{}
	authorizer := &fakeAuthorizer{err: authz.ErrDenied}
	router := gin.New()
	RegisterRoutes(router, NewHandler(service.NewUserService(repo), authorizer))

	response := performCreateUserRequest(router, `{"email":"ada@example.com","name":"Ada"}`)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
	}
	if repo.created != nil {
		t.Fatal("repository received a user after authorization was denied")
	}
}

func TestCreateUserAuthorizationUnavailableDoesNotCreateUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &fakeUserRepository{}
	authorizer := &fakeAuthorizer{err: errors.New("authz unavailable")}
	router := gin.New()
	RegisterRoutes(router, NewHandler(service.NewUserService(repo), authorizer))

	response := performCreateUserRequest(router, `{"email":"ada@example.com","name":"Ada"}`)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
	if repo.created != nil {
		t.Fatal("repository received a user when authorization was unavailable")
	}
}

func TestCreateUserMissingIdentityIsUnauthorized(t *testing.T) {
	repo := &fakeUserRepository{}
	request := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{"email":"ada@example.com","name":"Ada"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	newTestRouter(repo).ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
	if repo.created != nil {
		t.Fatal("repository received a user without trusted identity")
	}
}

func decodeResponse(t *testing.T, response *httptest.ResponseRecorder, destination any) {
	t.Helper()
	if err := json.Unmarshal(response.Body.Bytes(), destination); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

func mapsEqual(got, want map[string]string) bool {
	if len(got) != len(want) {
		return false
	}
	for key, wantValue := range want {
		if got[key] != wantValue {
			return false
		}
	}
	return true
}
