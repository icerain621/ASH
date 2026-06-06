package api

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ash-repwiki/ash/internal/config"
	"github.com/ash-repwiki/ash/internal/secrets"
	"github.com/ash-repwiki/ash/internal/store"
)

var secretNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.-]{0,127}$`)

type createSecretRequest struct {
	Name        string         `json:"name" binding:"required"`
	Value       string         `json:"value" binding:"required"`
	Description string         `json:"description,omitempty"`
	Scope       map[string]any `json:"scope,omitempty"`
	SpaceID     string         `json:"spaceId,omitempty"`
}

type rotateSecretRequest struct {
	Value       string `json:"value" binding:"required"`
	Description string `json:"description,omitempty"`
}

// ListSecrets godoc
// @Summary List space secrets
// @Tags secrets
// @Produce json
// @Success 200 {object} SecretListResponse
// @Failure 403 {object} APIErrorResponse
// @Router /api/v1/secrets [get]
func (h *Handler) listSecrets(c *gin.Context) {
	space := currentSpace(c)
	if !h.requirePermission(c, permSecretRead, space) {
		return
	}
	var rows []store.SecretRecord
	if err := h.dbFor(c).Where("space_id = ?", space).Order("name asc").Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("SECRET_LIST_FAILED", err.Error()))
		return
	}
	items := make([]SecretResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, secretResponse(row))
	}
	c.JSON(http.StatusOK, SecretListResponse{Items: items})
}

// CreateSecret godoc
// @Summary Create a space secret
// @Tags secrets
// @Accept json
// @Produce json
// @Param body body createSecretRequest true "secret"
// @Success 201 {object} SecretResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 403 {object} APIErrorResponse
// @Failure 409 {object} APIErrorResponse
// @Router /api/v1/secrets [post]
func (h *Handler) createSecret(c *gin.Context) {
	var req createSecretRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	space := firstNonEmptyAPI(req.SpaceID, currentSpace(c))
	if !h.requireTargetSpace(c, space) {
		return
	}
	if !h.requirePermission(c, permSecretWrite, space) {
		return
	}
	name := strings.TrimSpace(req.Name)
	if !validSecretName(name) {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_SECRET_NAME", "secret name must match ^[A-Za-z_][A-Za-z0-9_.-]{0,127}$"))
		return
	}
	var existing store.SecretRecord
	err := h.dbFor(c).Where("space_id = ? AND name = ?", space, name).First(&existing).Error
	if err == nil {
		c.JSON(http.StatusConflict, errorBody("SECRET_ALREADY_EXISTS", "secret already exists in this space"))
		return
	}
	if err != gorm.ErrRecordNotFound {
		c.JSON(http.StatusInternalServerError, errorBody("SECRET_LOOKUP_FAILED", err.Error()))
		return
	}
	ciphertext, digest, err := secrets.Seal(req.Value, config.Load().SecretKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("SECRET_ENCRYPT_FAILED", err.Error()))
		return
	}
	scopeJSON, err := marshalSecretScope(req.Scope)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_SECRET_SCOPE", err.Error()))
		return
	}
	now := time.Now().UTC()
	row := store.SecretRecord{
		ID:              "sec_" + uuid.NewString(),
		SpaceID:         space,
		Name:            name,
		Description:     strings.TrimSpace(req.Description),
		Status:          "active",
		ScopeJSON:       scopeJSON,
		ValueCiphertext: ciphertext,
		ValueDigest:     digest,
		CreatedBy:       currentActor(c),
		UpdatedBy:       currentActor(c),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := h.dbFor(c).Create(&row).Error; err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("SECRET_CREATE_FAILED", err.Error()))
		return
	}
	_ = h.dbFor(c).Create(auditRow(space, currentActor(c), "secret.created", map[string]any{
		"secretId": row.ID, "name": row.Name, "digest": row.ValueDigest,
	}))
	c.JSON(http.StatusCreated, secretResponse(row))
}

// RotateSecret godoc
// @Summary Rotate a secret value
// @Tags secrets
// @Accept json
// @Produce json
// @Param secretId path string true "secret id"
// @Param body body rotateSecretRequest true "new secret value"
// @Success 200 {object} SecretResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 403 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Router /api/v1/secrets/{secretId}/rotate [post]
func (h *Handler) rotateSecret(c *gin.Context) {
	var req rotateSecretRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	row, ok := h.secretByID(c, c.Param("secretId"), permSecretWrite)
	if !ok {
		return
	}
	ciphertext, digest, err := secrets.Seal(req.Value, config.Load().SecretKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("SECRET_ENCRYPT_FAILED", err.Error()))
		return
	}
	row.ValueCiphertext = ciphertext
	row.ValueDigest = digest
	row.UpdatedBy = currentActor(c)
	row.UpdatedAt = time.Now().UTC()
	if strings.TrimSpace(req.Description) != "" {
		row.Description = strings.TrimSpace(req.Description)
	}
	if err := h.dbFor(c).Save(&row).Error; err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("SECRET_ROTATE_FAILED", err.Error()))
		return
	}
	_ = h.dbFor(c).Create(auditRow(row.SpaceID, currentActor(c), "secret.rotated", map[string]any{
		"secretId": row.ID, "name": row.Name, "digest": row.ValueDigest,
	}))
	c.JSON(http.StatusOK, secretResponse(row))
}

// DeleteSecret godoc
// @Summary Delete a secret
// @Tags secrets
// @Produce json
// @Param secretId path string true "secret id"
// @Success 204
// @Failure 403 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Router /api/v1/secrets/{secretId} [delete]
func (h *Handler) deleteSecret(c *gin.Context) {
	row, ok := h.secretByID(c, c.Param("secretId"), permSecretWrite)
	if !ok {
		return
	}
	if err := h.dbFor(c).Delete(&row).Error; err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("SECRET_DELETE_FAILED", err.Error()))
		return
	}
	_ = h.dbFor(c).Create(auditRow(row.SpaceID, currentActor(c), "secret.deleted", map[string]any{
		"secretId": row.ID, "name": row.Name, "digest": row.ValueDigest,
	}))
	c.Status(http.StatusNoContent)
}

func (h *Handler) secretByID(c *gin.Context, id, permission string) (store.SecretRecord, bool) {
	var row store.SecretRecord
	if err := h.dbFor(c).First(&row, "id = ?", id).Error; err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, errorBody("SECRET_NOT_FOUND", "secret not found"))
		return row, false
	}
	if !h.requireRequestSpace(c, row.SpaceID) {
		return row, false
	}
	if !h.requirePermission(c, permission, row.SpaceID) {
		return row, false
	}
	return row, true
}

func validSecretName(name string) bool {
	return secretNamePattern.MatchString(name)
}

func marshalSecretScope(scope map[string]any) (string, error) {
	if scope == nil {
		scope = map[string]any{}
	}
	raw, err := json.Marshal(scope)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func secretResponse(row store.SecretRecord) SecretResponse {
	scope := map[string]any{}
	_ = json.Unmarshal([]byte(row.ScopeJSON), &scope)
	return SecretResponse{
		ID:            row.ID,
		SpaceID:       row.SpaceID,
		Name:          row.Name,
		Description:   row.Description,
		Status:        row.Status,
		Scope:         scope,
		ValueDigest:   row.ValueDigest,
		RedactedValue: "********",
		CreatedBy:     row.CreatedBy,
		UpdatedBy:     row.UpdatedBy,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
		LastUsedAt:    row.LastUsedAt,
	}
}
