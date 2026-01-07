package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"aidanwoods.dev/go-paseto"
	"github.com/mtsgn/mtsgn-system-gateway-svc/internal/server"
	"github.com/mtsgn/mtsgn-system-gateway-svc/pkg/config"
)

func (p *ProxyHandler) authorizationMiddleware(w http.ResponseWriter, r *http.Request, cfg *config.Config, service *server.ServiceConfig) error {
	if service == nil {
		return errors.New("service not found")
	}

	if service.SkipAuth {
		return nil
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return errors.New("authorization header is required")
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	tokenString = strings.TrimSpace(tokenString)

	parser := paseto.NewParser()

	publicKey, err := paseto.NewV4AsymmetricPublicKeyFromHex(cfg.Auth.JWTSecret)
	if err != nil {
		return fmt.Errorf("Invalid public key configuration: %v", err)
	}

	token, err := parser.ParseV4Public(publicKey, tokenString, nil)
	if err != nil {
		return fmt.Errorf("Invalid token: %v", err)
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(token.ClaimsJSON(), &claims); err != nil {
		return fmt.Errorf("Failed to parse token claims: %v", err)
	}

	if userID, exists := claims["userId"]; exists {
		r.Header.Set("X-User-ID", fmt.Sprintf("%v", userID))
	}
	if admin, exists := claims["isAdmin"]; exists {
		r.Header.Set("X-Is-Admin", fmt.Sprintf("%v", admin))
	}
	if issuedAt, exists := claims["iat"]; exists {
		r.Header.Set("X-Issued-At", fmt.Sprintf("%v", issuedAt))
	}
	if sessionId, exists := claims["sessionId"]; exists {
		r.Header.Set("X-Session-ID", fmt.Sprintf("%v", sessionId))
	}
	if customClaims, exists := claims["customClaims"]; exists {
		r.Header.Set("X-Custom-Claims", fmt.Sprintf("%v", customClaims))
	}
	if expiresAt, exists := claims["exp"]; exists {
		r.Header.Set("X-Exp", fmt.Sprintf("%v", expiresAt))
	}
	return nil
}
