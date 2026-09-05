package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/haidodev/user-service/api/generated"
	"github.com/haidodev/user-service/internal/service"
	"github.com/oapi-codegen/runtime/types"
)

// real handler should implement the generated ServerInterface
type Handler struct {
	userService *service.UserService
}

// constructor
func NewHandler(userService *service.UserService) *Handler {
	return &Handler{
		userService: userService,
	}
}

// methods

func (h *Handler) CreateUser(c *gin.Context) {
	var req generated.CreateUserRequest
	// form binding and validation
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

	ctx := c.Request.Context()
	user, err := h.userService.CreateUser(ctx, string(req.Email), req.Name)
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
		http.StatusCreated,
		generated.User{
			Id:    types.UUID(user.Id),
			Email: types.Email(user.Email),
			Name:  user.Name,
		},
	)
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
