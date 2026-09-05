package http

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/haidodev/user-service/api/generated"
	"github.com/haidodev/user-service/internal/repository"
	"github.com/haidodev/user-service/internal/service"
	"github.com/oapi-codegen/runtime/types"
)

// real handler should implement the generated ServerInterface
type Handler struct {
	userService *service.UserService
}

type createUserRequest struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

var requestValidator = validator.New()

// constructor
func NewHandler(userService *service.UserService) *Handler {
	return &Handler{
		userService: userService,
	}
}

// methods

func (h *Handler) CreateUser(c *gin.Context) {
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
	user, err := h.userService.CreateUser(ctx, req.Email, name)
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
			Id:    types.UUID(user.Id),
			Email: types.Email(user.Email),
			Name:  user.Name,
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

func (h *Handler) GetUser(c *gin.Context, id types.UUID) {
	ctx := c.Request.Context()
	user, err := h.userService.GetUser(ctx, uuid.UUID(id))
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
	c.JSON(
		http.StatusOK,
		generated.User{
			Id:    types.UUID(user.Id),
			Email: types.Email(user.Email),
			Name:  user.Name,
		},
	)
}

func (h *Handler) ListUsers(c *gin.Context) {
	users, err := h.userService.ListUsers(c.Request.Context())
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
