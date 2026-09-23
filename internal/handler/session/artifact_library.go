package session

import (
	"net/http"
	"strings"

	"github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/errors"
	"github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/logger"
	"github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/types"
	"github.com/gin-gonic/gin"
)

// ListArtifactLibrary godoc
// @Summary      List my artifacts
// @Description  List skill-generated files across every session visible to the current user; only the latest version of each file is returned (storage URL excluded)
// @Tags         Sessions
// @Produce      json
// @Param        keyword     query  string  false  "Filter by file name"
// @Param        file_types  query  string  false  "Comma-separated extensions, e.g. .pdf,.pptx"
// @Param        page        query  int     false  "Page number"
// @Param        page_size   query  int     false  "Items per page (max 100)"
// @Success      200  {object}  map[string]interface{}
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /artifacts [get]
//
// Items carry session_id, message_id and index so the client downloads
// through the per-session endpoint, which re-runs the ownership check.
func (h *Handler) ListArtifactLibrary(c *gin.Context) {
	ctx := c.Request.Context()

	var pagination types.Pagination
	if err := c.ShouldBindQuery(&pagination); err != nil {
		_ = c.Error(errors.NewBadRequestError(err.Error()))
		return
	}
	var fileTypes []string
	if raw := c.Query("file_types"); raw != "" {
		fileTypes = strings.Split(raw, ",")
	}

	result, err := h.messageService.ListArtifactLibrary(ctx, &types.ArtifactLibraryQuery{
		Keyword:   c.Query("keyword"),
		FileTypes: fileTypes,
		Page:      pagination.Page,
		PageSize:  pagination.PageSize,
	})
	if err != nil {
		logger.Errorf(ctx, "list artifact library failed: %v", err)
		_ = c.Error(errors.NewInternalServerError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"data":      result.Data,
		"total":     result.Total,
		"page":      result.Page,
		"page_size": result.PageSize,
	})
}
