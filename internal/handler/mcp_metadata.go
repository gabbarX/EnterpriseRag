package handler

import (
	"context"
	stderrors "errors"
	"net/http"

	"github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/errors"
	"github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/logger"
	"github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/types"
	"github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/types/interfaces"
	secutils "github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/utils"
	"github.com/gin-gonic/gin"
)

// GetMCPMetadata godoc
// @Summary      Read the saved MCP tool catalog
// @Description  Reads from the database only and does not connect upstream. data is null when nothing has been synced yet; stale is true after the connection configuration changes. OAuth catalogs are isolated per authorised subject.
// @Tags         MCP Services
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "MCP service ID"
// @Success      200  {object}  map[string]interface{}  "Catalog snapshot"
// @Failure      400  {object}  errors.AppError         "Invalid request parameters"
// @Failure      401  {object}  errors.AppError         "OAuth catalog is missing an authorised subject"
// @Failure      404  {object}  errors.AppError         "Service not found"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /mcp-services/{id}/metadata [get]
func (h *MCPServiceHandler) GetMCPMetadata(c *gin.Context) { h.mcpMetadata(c, false) }

// RefreshMCPMetadata godoc
// @Summary      Sync the MCP tool catalog
// @Description  Explicitly connects upstream and atomically replaces the full catalog. OAuth services write the snapshot of the current user and may be called by Viewer and above; static authentication writes the tenant-shared snapshot and requires Admin.
// @Tags         MCP Services
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "MCP service ID"
// @Success      200  {object}  map[string]interface{}  "Catalog snapshot after the sync"
// @Failure      400  {object}  errors.AppError         "Catalog is incomplete or failed validation"
// @Failure      401  {object}  errors.AppError         "OAuth catalog is missing an authorised subject"
// @Failure      403  {object}  errors.AppError         "Static authentication catalog requires an administrator to refresh"
// @Failure      404  {object}  errors.AppError         "Service not found"
// @Failure      409  {object}  errors.AppError         "Connection configuration changed during the refresh"
// @Failure      503  {object}  errors.AppError         "Metadata store is unavailable"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /mcp-services/{id}/metadata/refresh [post]
func (h *MCPServiceHandler) RefreshMCPMetadata(c *gin.Context) { h.mcpMetadata(c, true) }

func (h *MCPServiceHandler) mcpMetadata(c *gin.Context, refresh bool) {
	ctx := c.Request.Context()
	tenant := c.GetUint64(types.TenantIDContextKey.String())
	if tenant == 0 {
		_ = c.Error(errors.NewBadRequestError("Workspace ID cannot be empty"))
		return
	}
	svc, ok := h.mcpServiceService.(interfaces.MCPMetadataService)
	if !ok {
		_ = c.Error(errors.NewServiceUnavailableError("MCP metadata storage is unavailable"))
		return
	}
	id := c.Param("id")
	var snapshot *types.MCPMetadata
	var err error
	if refresh {
		service, getErr := h.mcpServiceService.GetMCPServiceByID(ctx, tenant, id)
		if getErr != nil || service == nil {
			logger.ErrorWithFields(ctx, getErr, map[string]interface{}{
				"service_id": secutils.SanitizeForLog(id),
				"refresh":    true,
			})
			_ = c.Error(mcpMetadataAppError(types.ErrMCPServiceNotFound, true))
			return
		}
		if !service.AuthConfig.IsOAuth() && !mayWriteSharedMCPMetadata(ctx) {
			_ = c.Error(errors.NewForbiddenError("Refreshing a shared MCP directory requires an administrator"))
			return
		}
		snapshot, err = svc.RefreshMCPMetadata(ctx, tenant, id)
	} else {
		snapshot, err = svc.GetMCPMetadata(ctx, tenant, id)
	}
	if err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{
			"service_id": secutils.SanitizeForLog(id),
			"refresh":    refresh,
		})
		_ = c.Error(mcpMetadataAppError(err, refresh))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": snapshot})
}

// mayWriteSharedMCPMetadata is the extra gate for static-auth catalogs. The
// route stays Viewer+ so OAuth users can persist their own snapshot after
// authorizing in chat. API keys already passed manage-MCP; JWT callers need Admin.
func mayWriteSharedMCPMetadata(ctx context.Context) bool {
	if _, ok := types.TenantAPIKeyScopeFromContext(ctx); ok {
		return true
	}
	if types.IsSystemAdminFromContext(ctx) {
		return true
	}
	return types.CallerFromContext(ctx).Role.HasPermission(types.TenantRoleAdmin)
}

func mcpMetadataAppError(err error, refresh bool) *errors.AppError {
	switch {
	case stderrors.Is(err, types.ErrMCPServiceNotFound):
		return errors.NewNotFoundError("MCP service not found")
	case stderrors.Is(err, types.ErrMCPOAuthPrincipalRequired):
		return errors.NewUnauthorizedError("OAuth metadata requires an authenticated user")
	case stderrors.Is(err, types.ErrMCPMetadataStorage):
		return errors.NewServiceUnavailableError("MCP metadata storage is unavailable")
	case stderrors.Is(err, types.ErrMCPMetadataConnectionChanged):
		return errors.NewConflictError("MCP connection changed during refresh; save the configuration and sync again")
	case stderrors.Is(err, types.ErrMCPMetadataTooLarge), stderrors.Is(err, types.ErrMCPMetadataInvalidTools):
		return errors.NewBadRequestError("MCP directory is invalid or too large")
	default:
		if refresh {
			return errors.NewBadRequestError("Failed to refresh MCP tools. Check the connection and try again.")
		}
		return errors.NewInternalServerError("Failed to read MCP metadata")
	}
}
