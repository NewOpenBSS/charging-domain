package keycloak

import (
	"context"
	"fmt"
	"go-ocs/internal/auth/config"

	"github.com/Nerzal/gocloak/v13"
)

// UserService provides access to the Keycloak admin API for fetching
// user attributes and role details that are not embedded in the JWT.
type UserService struct {
	gocloak *gocloak.GoCloak
	config  config.KeycloakConfig
	realm   string
}

// NewUserService creates a UserService using admin credentials from config.
func NewUserService(cfg config.KeycloakConfig) *UserService {
	gc := gocloak.NewClient(cfg.IssuerURL)
	return &UserService{
		gocloak: gc,
		config:  cfg,
		realm:   extractRealm(cfg.IssuerURL),
	}
}

// GetUserAttributes fetches the attributes map for a given Keycloak user ID.
func (s *UserService) GetUserAttributes(ctx context.Context, adminToken, userID string) (map[string][]string, error) {
	user, err := s.gocloak.GetUserByID(ctx, adminToken, s.realm, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user %s: %w", userID, err)
	}

	if user.Attributes == nil {
		return map[string][]string{}, nil
	}

	return *user.Attributes, nil
}

// GetRoleAttributes fetches the attributes defined on a Keycloak role.
func (s *UserService) GetRoleAttributes(ctx context.Context, adminToken, roleName string) (map[string][]string, error) {
	role, err := s.gocloak.GetRealmRole(ctx, adminToken, s.realm, roleName)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch role %s: %w", roleName, err)
	}

	if role.Attributes == nil {
		return map[string][]string{}, nil
	}

	return *role.Attributes, nil
}
