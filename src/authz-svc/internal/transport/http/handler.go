package http

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/haidodev/authz-service/internal/authz"
)

type Handler struct {
	engine            authz.Engine
	bootstrapOrg      uuid.UUID
	bootstrapUser     uuid.UUID
	provisioningToken string
}

func NewHandler(engine authz.Engine, bootstrapOrg, bootstrapUser uuid.UUID, provisioningToken ...string) *Handler {
	token := ""
	if len(provisioningToken) > 0 {
		token = provisioningToken[0]
	}
	return &Handler{engine: engine, bootstrapOrg: bootstrapOrg, bootstrapUser: bootstrapUser, provisioningToken: token}
}

func (h *Handler) Register(router gin.IRouter) {
	router.GET("/health/live", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.GET("/health/ready", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.POST("/v1/check", h.Check)
	router.POST("/v1/internal/organizations/:organizationId/bootstrap-admin/:userId", h.BootstrapAdmin)
	router.PUT("/v1/organizations/:organizationId/users/:userId/roles/admin", h.AssignAdmin)
	router.DELETE("/v1/organizations/:organizationId/users/:userId/roles/admin", h.RevokeAdmin)
	router.POST("/v1/organizations/:organizationId/roles", h.CreateRole)
	router.PUT("/v1/organizations/:organizationId/roles/:roleId/users/:userId", h.AssignRole)
	router.DELETE("/v1/organizations/:organizationId/roles/:roleId/users/:userId", h.RevokeRole)
	router.PUT("/v1/organizations/:organizationId/roles/:roleId/permissions/:permission", h.GrantRolePermission)
	router.DELETE("/v1/organizations/:organizationId/roles/:roleId/permissions/:permission", h.RevokeRolePermission)
}

// BootstrapAdmin is a private service-to-service endpoint used only while an
// organization is created. It avoids giving the identity service general role
// management rights.
func (h *Handler) BootstrapAdmin(c *gin.Context) {
	if h.provisioningToken == "" || c.GetHeader("X-Internal-Token") != h.provisioningToken {
		writeError(c, http.StatusUnauthorized, "unauthenticated", "invalid internal token")
		return
	}
	organizationID, err := uuid.Parse(c.Param("organizationId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", "organizationId must be a UUID")
		return
	}
	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", "userId must be a UUID")
		return
	}
	if err := h.engine.AssignAdmin(c.Request.Context(), userID, organizationID); err != nil {
		writeEngineError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
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

func (h *Handler) CreateRole(c *gin.Context) {
	_, organizationID, ok := h.requireOrganizationAdmin(c)
	if !ok {
		return
	}
	roleID := uuid.New()
	if err := h.engine.CreateRole(c.Request.Context(), organizationID, roleID); err != nil {
		writeEngineError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": roleID})
}

func (h *Handler) AssignRole(c *gin.Context) {
	roleID, _, ok := h.customRoleRequest(c)
	if !ok {
		return
	}
	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", "userId must be a UUID")
		return
	}
	if err := h.engine.AssignRole(c.Request.Context(), userID, roleID); err != nil {
		writeEngineError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) RevokeRole(c *gin.Context) {
	roleID, _, ok := h.customRoleRequest(c)
	if !ok {
		return
	}
	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", "userId must be a UUID")
		return
	}
	if err := h.engine.RevokeRole(c.Request.Context(), userID, roleID); err != nil {
		writeEngineError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) GrantRolePermission(c *gin.Context) {
	roleID, organizationID, permission, ok := h.rolePermissionRequest(c)
	if !ok {
		return
	}
	if err := h.engine.GrantRolePermission(c.Request.Context(), organizationID, roleID, permission); err != nil {
		writeEngineError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) RevokeRolePermission(c *gin.Context) {
	roleID, organizationID, permission, ok := h.rolePermissionRequest(c)
	if !ok {
		return
	}
	if err := h.engine.RevokeRolePermission(c.Request.Context(), organizationID, roleID, permission); err != nil {
		writeEngineError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) roleRequest(c *gin.Context) (uuid.UUID, uuid.UUID, uuid.UUID, bool) {
	actorID, organizationID, ok := h.requireOrganizationAdmin(c)
	if !ok {
		return uuid.Nil, uuid.Nil, uuid.Nil, false
	}
	targetID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", "userId must be a UUID")
		return uuid.Nil, uuid.Nil, uuid.Nil, false
	}
	return actorID, organizationID, targetID, true
}

func (h *Handler) requireOrganizationAdmin(c *gin.Context) (uuid.UUID, uuid.UUID, bool) {
	actorID, err := uuid.Parse(c.GetHeader("X-User-ID"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", "X-User-ID must be a UUID")
		return uuid.Nil, uuid.Nil, false
	}
	organizationID, err := uuid.Parse(c.Param("organizationId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", "organizationId must be a UUID")
		return uuid.Nil, uuid.Nil, false
	}
	allowed, err := h.engine.Check(c.Request.Context(), actorID, organizationID, authz.PermissionManageRoles)
	if err != nil {
		writeEngineError(c, err)
		return uuid.Nil, uuid.Nil, false
	}
	if !allowed {
		writeError(c, http.StatusForbidden, "permission_denied", "organization admin permission is required")
		return uuid.Nil, uuid.Nil, false
	}
	return actorID, organizationID, true
}

func (h *Handler) customRoleRequest(c *gin.Context) (uuid.UUID, uuid.UUID, bool) {
	_, organizationID, ok := h.requireOrganizationAdmin(c)
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}
	roleID, err := uuid.Parse(c.Param("roleId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", "roleId must be a UUID")
		return uuid.Nil, uuid.Nil, false
	}
	exists, err := h.engine.RoleExists(c.Request.Context(), organizationID, roleID)
	if err != nil {
		writeEngineError(c, err)
		return uuid.Nil, uuid.Nil, false
	}
	if !exists {
		writeError(c, http.StatusNotFound, "not_found", "role not found in organization")
		return uuid.Nil, uuid.Nil, false
	}
	return roleID, organizationID, true
}

func (h *Handler) rolePermissionRequest(c *gin.Context) (uuid.UUID, uuid.UUID, authz.Permission, bool) {
	roleID, organizationID, ok := h.customRoleRequest(c)
	if !ok {
		return uuid.Nil, uuid.Nil, "", false
	}
	permission := authz.Permission(c.Param("permission"))
	if !permission.AssignableToCustomRole() {
		writeError(c, http.StatusBadRequest, "invalid_request", "unsupported custom role permission")
		return uuid.Nil, uuid.Nil, "", false
	}
	return roleID, organizationID, permission, true
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
