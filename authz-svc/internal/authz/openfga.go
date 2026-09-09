package authz

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

var ErrTupleAlreadyExists = errors.New("OpenFGA tuple already exists")

const authorizationModel = `{"schema_version":"1.1","type_definitions":[{"type":"user"},{"type":"role","relations":{"organization":{"this":{}},"assignee":{"this":{}}},"metadata":{"relations":{"organization":{"directly_related_user_types":[{"type":"organization"}]},"assignee":{"directly_related_user_types":[{"type":"user"}]}}}},{"type":"organization","relations":{"admin":{"this":{}},"manage_roles":{"computedUserset":{"relation":"admin"}},"list_users":{"union":{"child":[{"computedUserset":{"relation":"admin"}},{"this":{}}]}},"create_user":{"union":{"child":[{"computedUserset":{"relation":"admin"}},{"this":{}}]}}},"metadata":{"relations":{"admin":{"directly_related_user_types":[{"type":"user"}]},"list_users":{"directly_related_user_types":[{"type":"role","relation":"assignee"}]},"create_user":{"directly_related_user_types":[{"type":"role","relation":"assignee"}]}}}}]}`

type Config struct {
	APIURL                  string
	StoreName               string
	BootstrapOrganizationID uuid.UUID
	BootstrapAdminUserID    uuid.UUID
}

type OpenFGA struct {
	apiURL        string
	storeName     string
	bootstrapOrg  uuid.UUID
	bootstrapUser uuid.UUID
	httpClient    *http.Client
	mu            sync.RWMutex
	storeID       string
	modelID       string
}

func NewOpenFGA(cfg Config) *OpenFGA {
	return &OpenFGA{
		apiURL:        strings.TrimRight(cfg.APIURL, "/"),
		storeName:     cfg.StoreName,
		bootstrapOrg:  cfg.BootstrapOrganizationID,
		bootstrapUser: cfg.BootstrapAdminUserID,
		httpClient:    &http.Client{Timeout: 5 * time.Second},
	}
}

func (f *OpenFGA) Initialize(ctx context.Context) error {
	if f.apiURL == "" || f.storeName == "" || f.bootstrapOrg == uuid.Nil || f.bootstrapUser == uuid.Nil {
		return fmt.Errorf("%w: missing OpenFGA bootstrap configuration", ErrUnavailable)
	}
	storeID, err := f.findOrCreateStore(ctx)
	if err != nil {
		return err
	}
	modelID, err := f.writeModel(ctx, storeID)
	if err != nil {
		return err
	}
	f.mu.Lock()
	f.storeID, f.modelID = storeID, modelID
	f.mu.Unlock()
	if err := f.AssignAdmin(ctx, f.bootstrapUser, f.bootstrapOrg); err != nil && !errors.Is(err, ErrTupleAlreadyExists) {
		return err
	}
	return nil
}

func (f *OpenFGA) Check(ctx context.Context, subjectID, organizationID uuid.UUID, permission Permission) (bool, error) {
	if !permission.Valid() {
		return false, fmt.Errorf("invalid permission %q", permission)
	}
	storeID, modelID, ok := f.identifiers()
	if !ok {
		return false, fmt.Errorf("%w: client has not been initialized", ErrUnavailable)
	}
	body := map[string]any{
		"authorization_model_id": modelID,
		"tuple_key":              map[string]string{"user": f.user(subjectID), "relation": string(permission), "object": f.organization(organizationID)},
	}
	var response struct {
		Allowed bool `json:"allowed"`
	}
	if err := f.request(ctx, http.MethodPost, "/stores/"+storeID+"/check", body, &response); err != nil {
		return false, err
	}
	return response.Allowed, nil
}

func (f *OpenFGA) AssignAdmin(ctx context.Context, subjectID, organizationID uuid.UUID) error {
	return f.writeRelation(ctx, "writes", f.user(subjectID), "admin", f.organization(organizationID))
}

func (f *OpenFGA) RevokeAdmin(ctx context.Context, subjectID, organizationID uuid.UUID) error {
	return f.writeRelation(ctx, "deletes", f.user(subjectID), "admin", f.organization(organizationID))
}

func (f *OpenFGA) CreateRole(ctx context.Context, organizationID, roleID uuid.UUID) error {
	return f.writeRelation(ctx, "writes", f.organization(organizationID), "organization", f.role(roleID))
}

func (f *OpenFGA) RoleExists(ctx context.Context, organizationID, roleID uuid.UUID) (bool, error) {
	return f.checkRelation(ctx, f.organization(organizationID), "organization", f.role(roleID))
}

func (f *OpenFGA) AssignRole(ctx context.Context, subjectID, roleID uuid.UUID) error {
	return f.writeRelation(ctx, "writes", f.user(subjectID), "assignee", f.role(roleID))
}

func (f *OpenFGA) RevokeRole(ctx context.Context, subjectID, roleID uuid.UUID) error {
	return f.writeRelation(ctx, "deletes", f.user(subjectID), "assignee", f.role(roleID))
}

func (f *OpenFGA) GrantRolePermission(ctx context.Context, organizationID, roleID uuid.UUID, permission Permission) error {
	return f.writeRelation(ctx, "writes", f.roleUserset(roleID), string(permission), f.organization(organizationID))
}

func (f *OpenFGA) RevokeRolePermission(ctx context.Context, organizationID, roleID uuid.UUID, permission Permission) error {
	return f.writeRelation(ctx, "deletes", f.roleUserset(roleID), string(permission), f.organization(organizationID))
}

func (f *OpenFGA) writeRelation(ctx context.Context, operation, user, relation, object string) error {
	storeID, modelID, ok := f.identifiers()
	if !ok {
		return fmt.Errorf("%w: client has not been initialized", ErrUnavailable)
	}
	body := map[string]any{
		"authorization_model_id": modelID,
		operation:                map[string]any{"tuple_keys": []map[string]string{{"user": user, "relation": relation, "object": object}}},
	}
	return f.request(ctx, http.MethodPost, "/stores/"+storeID+"/write", body, nil)
}

func (f *OpenFGA) checkRelation(ctx context.Context, user, relation, object string) (bool, error) {
	storeID, modelID, ok := f.identifiers()
	if !ok {
		return false, fmt.Errorf("%w: client has not been initialized", ErrUnavailable)
	}
	body := map[string]any{
		"authorization_model_id": modelID,
		"tuple_key":              map[string]string{"user": user, "relation": relation, "object": object},
	}
	var response struct {
		Allowed bool `json:"allowed"`
	}
	if err := f.request(ctx, http.MethodPost, "/stores/"+storeID+"/check", body, &response); err != nil {
		return false, err
	}
	return response.Allowed, nil
}

func (f *OpenFGA) findOrCreateStore(ctx context.Context) (string, error) {
	var listed struct {
		Stores []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"stores"`
	}
	if err := f.request(ctx, http.MethodGet, "/stores", nil, &listed); err != nil {
		return "", err
	}
	for _, store := range listed.Stores {
		if store.Name == f.storeName {
			return store.ID, nil
		}
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := f.request(ctx, http.MethodPost, "/stores", map[string]string{"name": f.storeName}, &created); err != nil {
		return "", err
	}
	if created.ID == "" {
		return "", fmt.Errorf("%w: OpenFGA returned an empty store ID", ErrUnavailable)
	}
	return created.ID, nil
}

func (f *OpenFGA) writeModel(ctx context.Context, storeID string) (string, error) {
	var model any
	if err := json.Unmarshal([]byte(authorizationModel), &model); err != nil {
		return "", fmt.Errorf("decode embedded model: %w", err)
	}
	var response struct {
		ID string `json:"authorization_model_id"`
	}
	if err := f.request(ctx, http.MethodPost, "/stores/"+storeID+"/authorization-models", model, &response); err != nil {
		return "", err
	}
	if response.ID == "" {
		return "", fmt.Errorf("%w: OpenFGA returned an empty model ID", ErrUnavailable)
	}
	return response.ID, nil
}

func (f *OpenFGA) request(ctx context.Context, method, path string, requestBody any, responseBody any) error {
	var body bytes.Buffer
	if requestBody != nil {
		if err := json.NewEncoder(&body).Encode(requestBody); err != nil {
			return fmt.Errorf("encode OpenFGA request: %w", err)
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, f.apiURL+path, &body)
	if err != nil {
		return fmt.Errorf("create OpenFGA request: %w", err)
	}
	if requestBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	response, err := f.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		var failure struct {
			Message string `json:"message"`
		}
		_ = json.NewDecoder(response.Body).Decode(&failure)
		if strings.Contains(failure.Message, "tuple which already exists") {
			return fmt.Errorf("%w: %s", ErrTupleAlreadyExists, failure.Message)
		}
		if failure.Message != "" {
			return fmt.Errorf("%w: OpenFGA returned status %d: %s", ErrUnavailable, response.StatusCode, failure.Message)
		}
		return fmt.Errorf("%w: OpenFGA returned status %d", ErrUnavailable, response.StatusCode)
	}
	if responseBody != nil {
		if err := json.NewDecoder(response.Body).Decode(responseBody); err != nil {
			return fmt.Errorf("%w: decode OpenFGA response: %v", ErrUnavailable, err)
		}
	}
	return nil
}

func (f *OpenFGA) identifiers() (string, string, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.storeID, f.modelID, f.storeID != "" && f.modelID != ""
}

func (f *OpenFGA) user(id uuid.UUID) string         { return "user:" + id.String() }
func (f *OpenFGA) organization(id uuid.UUID) string { return "organization:" + id.String() }
func (f *OpenFGA) role(id uuid.UUID) string         { return "role:" + id.String() }
func (f *OpenFGA) roleUserset(id uuid.UUID) string  { return f.role(id) + "#assignee" }
