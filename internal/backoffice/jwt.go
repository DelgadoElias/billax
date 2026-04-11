package backoffice

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// GenerateToken creates a JWT token for a backoffice user
// Token includes user_id, tenant_id, email, and role
// TTL defaults to 24 hours
func GenerateToken(user User, secret string, ttl time.Duration) (string, error) {
	if secret == "" {
		return "", fmt.Errorf("generate token: secret cannot be empty")
	}
	if user.ID == uuid.Nil {
		return "", fmt.Errorf("generate token: user id cannot be nil")
	}
	if user.TenantID == uuid.Nil {
		return "", fmt.Errorf("generate token: tenant id cannot be nil")
	}

	claims := jwt.MapClaims{
		"user_id":   user.ID.String(),
		"tenant_id": user.TenantID.String(),
		"email":     user.Email,
		"role":      string(user.Role),
		"exp":       time.Now().Add(ttl).Unix(),
		"iat":       time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("generate token: signing: %w", err)
	}

	return tokenStr, nil
}

// ValidateToken validates a JWT token and returns the claims
func ValidateToken(tokenStr, secret string) (*JWTClaims, error) {
	if secret == "" {
		return nil, fmt.Errorf("validate token: secret cannot be empty")
	}
	if tokenStr == "" {
		return nil, fmt.Errorf("validate token: token cannot be empty")
	}

	token, err := jwt.ParseWithClaims(tokenStr, &jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("validate token: parsing: %w", err)
	}

	claims, ok := token.Claims.(*jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("validate token: invalid claims")
	}

	// Extract and validate claims
	userIDStr, ok := (*claims)["user_id"].(string)
	if !ok {
		return nil, fmt.Errorf("validate token: user_id claim missing or invalid type")
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, fmt.Errorf("validate token: parsing user_id: %w", err)
	}

	tenantIDStr, ok := (*claims)["tenant_id"].(string)
	if !ok {
		return nil, fmt.Errorf("validate token: tenant_id claim missing or invalid type")
	}
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return nil, fmt.Errorf("validate token: parsing tenant_id: %w", err)
	}

	email, ok := (*claims)["email"].(string)
	if !ok {
		return nil, fmt.Errorf("validate token: email claim missing or invalid type")
	}

	roleStr, ok := (*claims)["role"].(string)
	if !ok {
		return nil, fmt.Errorf("validate token: role claim missing or invalid type")
	}
	role := Role(roleStr)

	return &JWTClaims{
		UserID:   userID,
		TenantID: tenantID,
		Email:    email,
		Role:     role,
	}, nil
}
