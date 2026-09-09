package authz

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/google/uuid"
)

func TestOpenFGAInitializeSeedsAdminAndChecksPermission(t *testing.T) {
	var wroteSeed bool
	transport := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		var body any
		status := http.StatusOK
		switch r.URL.Path {
		case "/stores":
			if r.Method == http.MethodGet {
				body = map[string]any{"stores": []any{}}
				break
			}
			body = map[string]string{"id": "store-id"}
		case "/stores/store-id/authorization-models":
			body = map[string]string{"authorization_model_id": "model-id"}
		case "/stores/store-id/write":
			wroteSeed = true
		case "/stores/store-id/check":
			body = map[string]bool{"allowed": true}
		default:
			status = http.StatusNotFound
		}
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(encoded)), Request: r}, nil
	})

	organizationID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	adminID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	client := NewOpenFGA(Config{
		APIURL:                  "http://openfga.test",
		StoreName:               "test-store",
		BootstrapOrganizationID: organizationID,
		BootstrapAdminUserID:    adminID,
	})
	client.httpClient = &http.Client{Transport: transport}
	if err := client.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	if !wroteSeed {
		t.Fatal("Initialize() did not write the bootstrap admin tuple")
	}
	allowed, err := client.Check(context.Background(), adminID, organizationID, PermissionListUsers)
	if err != nil || !allowed {
		t.Fatalf("Check() = %v, %v; want true, nil", allowed, err)
	}
}

func TestOpenFGAInitializeAcceptsExistingBootstrapTuple(t *testing.T) {
	transport := roundTripperFunc(func(request *http.Request) (*http.Response, error) {
		var body any
		status := http.StatusOK
		switch request.URL.Path {
		case "/stores":
			if request.Method == http.MethodGet {
				body = map[string]any{"stores": []any{}}
			} else {
				body = map[string]string{"id": "store-id"}
			}
		case "/stores/store-id/authorization-models":
			body = map[string]string{"authorization_model_id": "model-id"}
		case "/stores/store-id/write":
			status = http.StatusBadRequest
			body = map[string]string{"message": "cannot write a tuple which already exists"}
		default:
			status = http.StatusNotFound
		}
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(encoded)), Request: request}, nil
	})

	client := NewOpenFGA(Config{
		APIURL:                  "http://openfga.test",
		StoreName:               "test-store",
		BootstrapOrganizationID: uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		BootstrapAdminUserID:    uuid.MustParse("11111111-1111-1111-1111-111111111111"),
	})
	client.httpClient = &http.Client{Transport: transport}
	if err := client.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize() error = %v, want nil for existing bootstrap tuple", err)
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }
