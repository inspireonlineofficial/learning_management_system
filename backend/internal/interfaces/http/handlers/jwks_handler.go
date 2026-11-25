package handlers

import (
	"lms-backend/internal/infrastructure/jwt"
	"net/http"
)

// JWKSHandler handles JWKS endpoint requests
type JWKSHandler struct {
	jwtService *jwt.JWTService
}

// NewJWKSHandler creates a new JWKS handler
func NewJWKSHandler(jwtService *jwt.JWTService) *JWKSHandler {
	return &JWKSHandler{
		jwtService: jwtService,
	}
}

// GetJWKS handles GET /v1/.well-known/jwks.json
//
// @Summary      Get JSON Web Key Set
// @Description  Returns the public keys used to verify JWT tokens issued by this server
// @Tags         auth
// @Produce      json
// @Success      200  {object}  object{keys=array}
// @Router       /v1/.well-known/jwks.json [get]
func (h *JWKSHandler) GetJWKS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	jwks, err := h.jwtService.GetJWKS()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve JWKS", nil)
		return
	}

	writeJSON(w, http.StatusOK, jwks)
}
