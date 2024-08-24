package user

type UserInfo struct {
	IsImpersonating      bool         `json:"isImpersonating"`
	IsHyphenInternal     bool         `json:"isHyphenInternal"`
	DecodedIdToken       TokenInfo    `json:"decodedIdToken"`
	AccessTokenExpiresIn int          `json:"accessTokenExpiresIn"`
	AccessTokenExpiresAt int          `json:"accessTokenExpiresAt"`
	DecodedToken         TokenInfo    `json:"decodedToken"`
	Memberships          []Membership `json:"memberships"`
}

type TokenInfo struct {
	EmailVerified     bool   `json:"email_verified"`
	Name              string `json:"name"`
	PreferredUsername string `json:"preferred_username"`
	GivenName         string `json:"given_name"`
	FamilyName        string `json:"family_name"`
	Email             string `json:"email"`
	Exp               int    `json:"exp,omitempty"`
	Iat               int    `json:"iat,omitempty"`
	Iss               string `json:"iss,omitempty"`
	Aud               string `json:"aud,omitempty"`
	Sub               string `json:"sub,omitempty"`
	Scope             string `json:"scope,omitempty"`
}

type Membership struct {
	ID                string                   `json:"id"`
	FirstName         string                   `json:"firstName"`
	LastName          string                   `json:"lastName"`
	Email             string                   `json:"email"`
	Nickname          string                   `json:"nickname"`
	ConnectedAccounts []map[string]interface{} `json:"connectedAccounts"`
	Organization      Organization             `json:"organization"`
	Teams             []map[string]interface{} `json:"teams"`
	Resources         []map[string]interface{} `json:"resources"`
	RoleMappings      []RoleMapping            `json:"roleMappings"`
}

type Organization struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type RoleMapping struct {
	ID                   string   `json:"id"`
	Type                 string   `json:"type"`
	EffectiveRoles       []string `json:"effectiveRoles"`
	EffectivePermissions []string `json:"effectivePermissions"`
}
