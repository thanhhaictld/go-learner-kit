package authz

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Permission string

const (
	PermissionListUsers   Permission = "list_users"
	PermissionCreateUsers Permission = "create_user"
)

var (
	ErrDenied      = errors.New("permission denied")
	ErrUnavailable = errors.New("authorization service unavailable")
)

type Authorizer interface {
	Check(ctx context.Context, subjectID, organizationID uuid.UUID, permission Permission) error
}

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Timeout: 3 * time.Second},
	}
}

type checkRequest struct {
	SubjectID      uuid.UUID  `json:"subjectId"`
	OrganizationID uuid.UUID  `json:"organizationId"`
	Permission     Permission `json:"permission"`
}

type checkResponse struct {
	Allowed bool `json:"allowed"`
}

func (c *Client) Check(ctx context.Context, subjectID, organizationID uuid.UUID, permission Permission) error {
	if c.baseURL == "" {
		return ErrUnavailable
	}
	body, err := json.Marshal(checkRequest{SubjectID: subjectID, OrganizationID: organizationID, Permission: permission})
	if err != nil {
		return fmt.Errorf("marshal authorization check: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/check", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create authorization check request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	response, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusForbidden {
		return ErrDenied
	}
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: status %d", ErrUnavailable, response.StatusCode)
	}

	var result checkResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return fmt.Errorf("%w: invalid response: %v", ErrUnavailable, err)
	}
	if !result.Allowed {
		return ErrDenied
	}
	return nil
}
