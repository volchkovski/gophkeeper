package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	apperrors "github.com/volchkovski/gophkeeper/internal/common/errors"
	"github.com/volchkovski/gophkeeper/internal/server/models"
	"github.com/volchkovski/gophkeeper/pkg/logger"
)

// CreateSecretRequest represents a create secret request.
type CreateSecretRequest struct {
	Type          string `json:"type" binding:"required"`
	Name          string `json:"name" binding:"required"`
	EncryptedData []byte `json:"encrypted_data" binding:"required"`
	Metadata      string `json:"metadata,omitempty"`
}

// UpdateSecretRequest represents an update secret request.
type UpdateSecretRequest struct {
	Type          string `json:"type" binding:"required"`
	Name          string `json:"name" binding:"required"`
	EncryptedData []byte `json:"encrypted_data" binding:"required"`
	Metadata      string `json:"metadata,omitempty"`
}

// SecretResponse represents a secret in API responses.
type SecretResponse struct {
	ID            string `json:"id"`
	Type          string `json:"type"`
	Name          string `json:"name"`
	EncryptedData []byte `json:"encrypted_data"`
	Metadata      string `json:"metadata,omitempty"`
	Version       int64  `json:"version"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

// SecretsListResponse represents a list of secrets.
type SecretsListResponse struct {
	Secrets []SecretResponse `json:"secrets"`
}

// SyncRequest represents a sync request.
type SyncRequest struct {
	Secrets []SyncSecretRequest `json:"secrets"`
}

// SyncSecretRequest represents a secret in sync request.
type SyncSecretRequest struct {
	ID            string `json:"id"`
	Type          string `json:"type"`
	Name          string `json:"name"`
	EncryptedData []byte `json:"encrypted_data"`
	Metadata      string `json:"metadata,omitempty"`
	Version       int64  `json:"version"`
	UpdatedAt     string `json:"updated_at"`
}

// SyncResponse represents a sync response.
type SyncResponse struct {
	UpdatedSecrets []SecretResponse `json:"updated_secrets"`
	Conflicts      []ConflictResponse `json:"conflicts,omitempty"`
}

// ConflictResponse represents a sync conflict.
type ConflictResponse struct {
	ClientVersion SecretResponse `json:"client_version"`
	ServerVersion SecretResponse `json:"server_version"`
}

// CreateSecret handles secret creation.
// @Summary Create a new secret
// @Tags secrets
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateSecretRequest true "Secret data"
// @Success 201 {object} SecretResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Router /secrets [post]
func (h *Handler) CreateSecret(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		respondWithError(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateSecretRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	secretType := models.SecretType(req.Type)
	if !secretType.IsValid() {
		respondWithError(c, http.StatusBadRequest, "invalid secret type")
		return
	}

	secret := &models.SecretData{
		Type:          secretType,
		Name:          req.Name,
		EncryptedData: req.EncryptedData,
		Metadata:      req.Metadata,
	}

	if err := h.secretService.Create(c.Request.Context(), userID, secret); err != nil {
		switch {
		case errors.Is(err, apperrors.ErrSecretNameExists):
			respondWithError(c, http.StatusConflict, "secret with this name already exists")
		case errors.Is(err, apperrors.ErrInvalidSecretType),
			errors.Is(err, apperrors.ErrInvalidSecretName),
			errors.Is(err, apperrors.ErrInvalidInput):
			respondWithError(c, http.StatusBadRequest, err.Error())
		default:
			h.logger.Error("create secret failed", logger.Error(err))
			respondWithError(c, http.StatusInternalServerError, "failed to create secret")
		}
		return
	}

	c.JSON(http.StatusCreated, secretToResponse(secret))
}

// GetSecret handles retrieving a single secret.
// @Summary Get a secret by ID
// @Tags secrets
// @Produce json
// @Security BearerAuth
// @Param id path string true "Secret ID"
// @Success 200 {object} SecretResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /secrets/{id} [get]
func (h *Handler) GetSecret(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		respondWithError(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	secretID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondWithError(c, http.StatusBadRequest, "invalid secret ID")
		return
	}

	secret, err := h.secretService.Get(c.Request.Context(), userID, secretID)
	if err != nil {
		switch {
		case errors.Is(err, apperrors.ErrSecretNotFound):
			respondWithError(c, http.StatusNotFound, "secret not found")
		case errors.Is(err, apperrors.ErrSecretAccessDenied):
			respondWithError(c, http.StatusForbidden, "access denied")
		default:
			h.logger.Error("get secret failed", logger.Error(err))
			respondWithError(c, http.StatusInternalServerError, "failed to get secret")
		}
		return
	}

	c.JSON(http.StatusOK, secretToResponse(secret))
}

// ListSecrets handles listing all user secrets.
// @Summary List all secrets
// @Tags secrets
// @Produce json
// @Security BearerAuth
// @Success 200 {object} SecretsListResponse
// @Failure 401 {object} ErrorResponse
// @Router /secrets [get]
func (h *Handler) ListSecrets(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		respondWithError(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	secrets, err := h.secretService.List(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("list secrets failed", logger.Error(err))
		respondWithError(c, http.StatusInternalServerError, "failed to list secrets")
		return
	}

	response := SecretsListResponse{
		Secrets: make([]SecretResponse, len(secrets)),
	}
	for i, secret := range secrets {
		response.Secrets[i] = secretToResponse(secret)
	}

	c.JSON(http.StatusOK, response)
}

// UpdateSecret handles secret updates.
// @Summary Update a secret
// @Tags secrets
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Secret ID"
// @Param request body UpdateSecretRequest true "Updated secret data"
// @Success 200 {object} SecretResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /secrets/{id} [put]
func (h *Handler) UpdateSecret(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		respondWithError(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	secretID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondWithError(c, http.StatusBadRequest, "invalid secret ID")
		return
	}

	var req UpdateSecretRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	secretType := models.SecretType(req.Type)
	if !secretType.IsValid() {
		respondWithError(c, http.StatusBadRequest, "invalid secret type")
		return
	}

	secret := &models.SecretData{
		ID:            secretID,
		Type:          secretType,
		Name:          req.Name,
		EncryptedData: req.EncryptedData,
		Metadata:      req.Metadata,
	}

	if err := h.secretService.Update(c.Request.Context(), userID, secret); err != nil {
		switch {
		case errors.Is(err, apperrors.ErrSecretNotFound):
			respondWithError(c, http.StatusNotFound, "secret not found")
		case errors.Is(err, apperrors.ErrSecretAccessDenied):
			respondWithError(c, http.StatusForbidden, "access denied")
		case errors.Is(err, apperrors.ErrSecretNameExists):
			respondWithError(c, http.StatusConflict, "secret with this name already exists")
		case errors.Is(err, apperrors.ErrInvalidSecretType),
			errors.Is(err, apperrors.ErrInvalidSecretName),
			errors.Is(err, apperrors.ErrInvalidInput):
			respondWithError(c, http.StatusBadRequest, err.Error())
		default:
			h.logger.Error("update secret failed", logger.Error(err))
			respondWithError(c, http.StatusInternalServerError, "failed to update secret")
		}
		return
	}

	// Get updated secret
	updated, err := h.secretService.Get(c.Request.Context(), userID, secretID)
	if err != nil {
		h.logger.Error("get updated secret failed", logger.Error(err))
		respondWithError(c, http.StatusInternalServerError, "failed to get updated secret")
		return
	}

	c.JSON(http.StatusOK, secretToResponse(updated))
}

// DeleteSecret handles secret deletion.
// @Summary Delete a secret
// @Tags secrets
// @Security BearerAuth
// @Param id path string true "Secret ID"
// @Success 204
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /secrets/{id} [delete]
func (h *Handler) DeleteSecret(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		respondWithError(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	secretID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondWithError(c, http.StatusBadRequest, "invalid secret ID")
		return
	}

	if err := h.secretService.Delete(c.Request.Context(), userID, secretID); err != nil {
		switch {
		case errors.Is(err, apperrors.ErrSecretNotFound):
			respondWithError(c, http.StatusNotFound, "secret not found")
		case errors.Is(err, apperrors.ErrSecretAccessDenied):
			respondWithError(c, http.StatusForbidden, "access denied")
		default:
			h.logger.Error("delete secret failed", logger.Error(err))
			respondWithError(c, http.StatusInternalServerError, "failed to delete secret")
		}
		return
	}

	c.Status(http.StatusNoContent)
}

// SyncSecrets handles secret synchronization.
// @Summary Sync secrets with server
// @Tags secrets
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body SyncRequest true "Secrets to sync"
// @Success 200 {object} SyncResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /secrets/sync [post]
func (h *Handler) SyncSecrets(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		respondWithError(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req SyncRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	// Convert request to models
	clientSecrets := make([]*models.SecretData, len(req.Secrets))
	for i, s := range req.Secrets {
		secretID, _ := uuid.Parse(s.ID)
		clientSecrets[i] = &models.SecretData{
			ID:            secretID,
			Type:          models.SecretType(s.Type),
			Name:          s.Name,
			EncryptedData: s.EncryptedData,
			Metadata:      s.Metadata,
			Version:       s.Version,
		}
	}

	result, err := h.secretService.Sync(c.Request.Context(), userID, clientSecrets)
	if err != nil {
		h.logger.Error("sync secrets failed", logger.Error(err))
		respondWithError(c, http.StatusInternalServerError, "failed to sync secrets")
		return
	}

	response := SyncResponse{
		UpdatedSecrets: make([]SecretResponse, len(result.UpdatedSecrets)),
	}
	for i, secret := range result.UpdatedSecrets {
		response.UpdatedSecrets[i] = secretToResponse(secret)
	}

	if len(result.Conflicts) > 0 {
		response.Conflicts = make([]ConflictResponse, len(result.Conflicts))
		for i, conflict := range result.Conflicts {
			response.Conflicts[i] = ConflictResponse{
				ClientVersion: secretToResponse(conflict.ClientVersion),
				ServerVersion: secretToResponse(conflict.ServerVersion),
			}
		}
	}

	c.JSON(http.StatusOK, response)
}

// secretToResponse converts a SecretData to SecretResponse.
func secretToResponse(s *models.SecretData) SecretResponse {
	return SecretResponse{
		ID:            s.ID.String(),
		Type:          string(s.Type),
		Name:          s.Name,
		EncryptedData: s.EncryptedData,
		Metadata:      s.Metadata,
		Version:       s.Version,
		CreatedAt:     s.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:     s.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

