package http

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/haidodev/user-service/api/generated"
	"github.com/haidodev/user-service/internal/authz"
	"github.com/haidodev/user-service/internal/repository"
	"github.com/haidodev/user-service/internal/service"
	"github.com/oapi-codegen/runtime/types"
	"gorm.io/gorm"
)

// real handler should implement the generated ServerInterface
type Handler struct {
	userService *service.UserService
	authorizer  authz.Authorizer
}

type createUserRequest struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

var requestValidator = validator.New()

// constructor
func NewHandler(userService *service.UserService, authorizer authz.Authorizer) *Handler {
	return &Handler{
		userService: userService,
		authorizer:  authorizer,
	}
}

// methods

func (h *Handler) CreateUser(c *gin.Context, _ generated.CreateUserParams) {
	_, organizationID, ok := h.authorize(c, authz.PermissionCreateUsers)
	if !ok {
		return
	}

	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(
			http.StatusBadRequest,
			generated.Error{
				Code:    "invalid_request",
				Message: err.Error(),
			},
		)
		return
	}

	name, fieldErrors := validateCreateUserRequest(req)
	if len(fieldErrors) > 0 {
		c.JSON(
			http.StatusBadRequest,
			generated.Error{
				Code:    "invalid_request",
				Message: "validation failed",
				Errors:  fieldErrorMap(fieldErrors),
			},
		)
		return
	}

	ctx := c.Request.Context()
	user, err := h.userService.CreateUser(ctx, organizationID, req.Email, name)
	if err != nil {
		if errors.Is(err, repository.ErrEmailAlreadyExists) {
			c.JSON(
				http.StatusConflict,
				generated.Error{
					Code:    "conflict",
					Message: "email already exists",
					Errors:  fieldErrorMap(map[string]string{"email": "already exists"}),
				},
			)
			return
		}
		c.JSON(
			http.StatusInternalServerError,
			generated.Error{
				Code:    "internal_error",
				Message: err.Error(),
			},
		)
		return
	}
	c.JSON(
		http.StatusCreated,
		generated.User{
			Id:        types.UUID(user.Id),
			Email:     types.Email(user.Email),
			Name:      user.Name,
			CreatedAt: user.CreatedAt,
		},
	)
}

func validateCreateUserRequest(req createUserRequest) (string, map[string]string) {
	fieldErrors := make(map[string]string)
	if req.Email == "" {
		fieldErrors["email"] = "is required"
	} else if err := requestValidator.Var(req.Email, "email"); err != nil {
		fieldErrors["email"] = "must be a valid email address"
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		fieldErrors["name"] = "must not be blank"
	}
	return name, fieldErrors
}

func fieldErrorMap(fields map[string]string) *map[string]string {
	return &fields
}

func (h *Handler) GetUser(c *gin.Context, id types.UUID, _ generated.GetUserParams) {
	_, organizationID, ok := h.authorize(c, authz.PermissionListUsers)
	if !ok {
		return
	}

	ctx := c.Request.Context()
	user, err := h.userService.GetUser(ctx, uuid.UUID(id), organizationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, generated.Error{Code: "not_found", Message: "user not found"})
			return
		}
		c.JSON(
			http.StatusInternalServerError,
			generated.Error{
				Code:    "internal_error",
				Message: err.Error(),
			},
		)
		return
	}
	c.JSON(
		http.StatusOK,
		generated.User{
			Id:        types.UUID(user.Id),
			Email:     types.Email(user.Email),
			Name:      user.Name,
			CreatedAt: user.CreatedAt,
		},
	)
}

func (h *Handler) ListUsers(c *gin.Context, _ generated.ListUsersParams) {
	_, organizationID, ok := h.authorize(c, authz.PermissionListUsers)
	if !ok {
		return
	}

	users, err := h.userService.ListUsers(c.Request.Context(), organizationID)
	if err != nil {
		c.JSON(
			http.StatusInternalServerError,
			generated.Error{
				Code:    "internal_error",
				Message: err.Error(),
			},
		)
		return
	}

	response := make([]generated.User, 0, len(users))
	for _, user := range users {
		response = append(response, generated.User{
			Id:        types.UUID(user.Id),
			Email:     types.Email(user.Email),
			Name:      user.Name,
			CreatedAt: user.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, response)
}

func (h *Handler) authorize(c *gin.Context, permission authz.Permission) (uuid.UUID, uuid.UUID, bool) {
	userID, err := requiredUUIDHeader(c, "X-User-ID")
	if err != nil {
		writeIdentityError(c, err)
		return uuid.Nil, uuid.Nil, false
	}
	organizationID, err := requiredUUIDHeader(c, "X-Organization-ID")
	if err != nil {
		writeIdentityError(c, err)
		return uuid.Nil, uuid.Nil, false
	}
	if h.authorizer == nil {
		c.JSON(http.StatusServiceUnavailable, generated.Error{Code: "authorization_unavailable", Message: "authorization service is unavailable"})
		return uuid.Nil, uuid.Nil, false
	}
	if err := h.authorizer.Check(c.Request.Context(), userID, organizationID, permission); err != nil {
		if errors.Is(err, authz.ErrDenied) {
			c.JSON(http.StatusForbidden, generated.Error{Code: "permission_denied", Message: "permission denied"})
			return uuid.Nil, uuid.Nil, false
		}
		c.JSON(http.StatusServiceUnavailable, generated.Error{Code: "authorization_unavailable", Message: "authorization service is unavailable"})
		return uuid.Nil, uuid.Nil, false
	}
	return userID, organizationID, true
}

var errMissingIdentityHeader = errors.New("missing identity header")

func requiredUUIDHeader(c *gin.Context, header string) (uuid.UUID, error) {
	value := c.GetHeader(header)
	if value == "" {
		return uuid.Nil, fmt.Errorf("%w: %s", errMissingIdentityHeader, header)
	}
	id, err := uuid.Parse(value)
	if err != nil || id.String() != value {
		return uuid.Nil, fmt.Errorf("invalid %s", header)
	}
	return id, nil
}

func writeIdentityError(c *gin.Context, err error) {
	if errors.Is(err, errMissingIdentityHeader) {
		c.JSON(http.StatusUnauthorized, generated.Error{Code: "unauthenticated", Message: "missing trusted identity headers"})
		return
	}
	c.JSON(http.StatusBadRequest, generated.Error{Code: "invalid_request", Message: err.Error()})
}
