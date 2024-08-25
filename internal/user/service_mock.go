package user

import (
	"github.com/Hyphen/cli/pkg/errors"
	"time"
)

// MockUserService is a mock implementation of UserServicer
type MockUserService struct {
	GetUserInformationFunc func() (UserInfo, error)
}

// Ensure MockUserService implements UserServicer
var _ UserServicer = (*MockUserService)(nil)

// GetUserInformation calls the mocked GetUserInformationFunc
func (m *MockUserService) GetUserInformation() (UserInfo, error) {
	if m.GetUserInformationFunc != nil {
		return m.GetUserInformationFunc()
	}
	return UserInfo{}, errors.New("GetUserInformation: not implemented")
}

// NewMockUserService creates a new instance of MockUserService with default behavior
func NewMockUserService() *MockUserService {
	return &MockUserService{
		GetUserInformationFunc: func() (UserInfo, error) {
			now := time.Now()
			return UserInfo{
				IsImpersonating:      false,
				IsHyphenInternal:     false,
				AccessTokenExpiresIn: 3600,
				AccessTokenExpiresAt: int(now.Add(1 * time.Hour).Unix()),
				DecodedIdToken: TokenInfo{
					EmailVerified:     true,
					Name:              "Mock User",
					PreferredUsername: "mockuser",
					GivenName:         "Mock",
					FamilyName:        "User",
					Email:             "mock@example.com",
					Exp:               int(now.Add(1 * time.Hour).Unix()),
					Iat:               int(now.Unix()),
					Iss:               "https://mock-issuer.com",
					Aud:               "mock-audience",
					Sub:               "mock-subject",
					Scope:             "openid profile email",
				},
				DecodedToken: TokenInfo{
					// Similar to DecodedIdToken, but you might want to differentiate if needed
				},
				Memberships: []Membership{
					{
						ID:        "mock-membership-id",
						FirstName: "Mock",
						LastName:  "User",
						Email:     "mock@example.com",
						Nickname:  "MockUser",
						ConnectedAccounts: []map[string]interface{}{
							{"provider": "github", "username": "mockuser"},
						},
						Organization: Organization{
							ID:   "mock-org-id",
							Name: "Mock Organization",
						},
						Teams: []map[string]interface{}{
							{"id": "mock-team-id", "name": "Mock Team"},
						},
						Resources: []map[string]interface{}{
							{"id": "mock-resource-id", "name": "Mock Resource"},
						},
						RoleMappings: []RoleMapping{
							{
								ID:                   "mock-role-mapping-id",
								Type:                 "user",
								EffectiveRoles:       []string{"user", "developer"},
								EffectivePermissions: []string{"read", "write"},
							},
						},
					},
				},
			}, nil
		},
	}
}
