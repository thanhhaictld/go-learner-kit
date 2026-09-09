package http

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/haidodev/authz-service/internal/authz"
)

type Handler struct {
	engine        authz.Engine
	bootstrapOrg  uuid.UUID
	bootstrapUser uuid.UUID
}

func NewHandler(engine authz.Engine, bootstrapOrg, bootstrapUser uuid.UUID) *Handler {
	return &Handler{engine: engine, bootstrapOrg: bootstrapOrg, bootstrapUser: bootstrapUser}
}

func (h *Handler) Register(router gin.IRouter) {
	router.GET("/health/live", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.GET("/health/ready", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.POST("/v1/check", h.Check)
	router.PUT("/v1/organizations/:organizationId/users/:userId/roles/admin", h.AssignAdmin)
	router.DELETE("/v1/organizations/:organizationId/users/:userId/roles/admin", h.RevokeAdmin)
}

type checkRequest struct {
	SubjectID      string `json:"subjectId"`
	OrganizationID string `json:"organizationId"`
	Permission     string `json:"permission"`
}

func (h *Handler) Check(c *gin.Context) {
	var request checkRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", "invalid check request")
		return
	}
	subjectID, err := uuid.Parse(request.SubjectID)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", "subjectId must be a UUID")
		return
	}
	organizationID, err := uuid.Parse(request.OrganizationID)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", "organizationId must be a UUID")
		return
	}
	permission := authz.Permission(request.Permission)
	if !permission.Valid() {
		writeError(c, http.StatusBadRequest, "invalid_request", "unsupported permission")
		return
	}
	allowed, err := h.engine.Check(c.Request.Context(), subjectID, organizationID, permission)
	if err != nil {
		writeEngineError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"allowed": allowed})
}

func (h *Handler) AssignAdmin(c *gin.Context) {
	_, organizationID, targetID, ok := h.roleRequest(c)
	if !ok {
		return
	}
	if err := h.engine.AssignAdmin(c.Request.Context(), targetID, organizationID); err != nil {
		writeEngineError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) RevokeAdmin(c *gin.Context) {
	_, organizationID, targetID, ok := h.roleRequest(c)
	if !ok {
		return
	}
	if organizationID == h.bootstrapOrg && targetID == h.bootstrapUser {
		writeError(c, http.StatusBadRequest, "protected_assignment", "the configured bootstrap admin cannot be revoked")
		return
	}
	if err := h.engine.RevokeAdmin(c.Request.Context(), targetID, organizationID); err != nil {
		writeEngineError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) roleRequest(c *gin.Context) (uuid.UUID, uuid.UUID, uuid.UUID, bool) {
	actorID, err := uuid.Parse(c.GetHeader("X-User-ID"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", "X-User-ID must be a UUID")
		return uuid.Nil, uuid.Nil, uuid.Nil, false
	}
	organizationID, err := uuid.Parse(c.Param("organizationId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", "organizationId must be a UUID")
		return uuid.Nil, uuid.Nil, uuid.Nil, false
	}
	targetID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", "userId must be a UUID")
		return uuid.Nil, uuid.Nil, uuid.Nil, false
	}
	allowed, err := h.engine.Check(c.Request.Context(), actorID, organizationID, authz.PermissionListUsers)
	if err != nil {
		writeEngineError(c, err)
		return uuid.Nil, uuid.Nil, uuid.Nil, false
	}
	if !allowed {
		writeError(c, http.StatusForbidden, "permission_denied", "organization admin permission is required")
		return uuid.Nil, uuid.Nil, uuid.Nil, false
	}
	return actorID, organizationID, targetID, true
}

func writeEngineError(c *gin.Context, err error) {
	if errors.Is(err, authz.ErrDenied) {
		writeError(c, http.StatusForbidden, "permission_denied", "permission denied")
		return
	}
	writeError(c, http.StatusServiceUnavailable, "authorization_unavailable", "authorization engine is unavailable")
}

func writeError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"code": code, "message": message})
}
