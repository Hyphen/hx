package models

type ConnectionEntity struct {
	Id   string `json:"id"`
	Type string `json:"type"`
	Name string `json:"name"`
}

type ConnectionOrganizationIntegration struct {
	Id   string `json:"id"`
	Type string `json:"type"`
}

type AzureContainerRegistryConfiguration struct {
	RegistryId   string `json:"registryId"`
	RegistryName string `json:"registryName"`
	TenantId     string `json:"tenantId"`
	Secrets      struct {
		Auth struct {
			Server            string `json:"server"`
			Username          string `json:"username"`
			EncryptedPassword string `json:"encryptedPassword"`
		} `json:"auth"`
	} `json:"secrets"`
}

type ContainerRegistry struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	Url  string `json:"url"`
	Auth struct {
		Server   string `json:"server"`
		Username string `json:"username"`
		Password string `json:"password"`
	} `json:"auth"`
}

type Connection[T AzureContainerRegistryConfiguration] struct {
	Id                      string                            `json:"id"`
	Type                    string                            `json:"type"`
	Entity                  ConnectionEntity                  `json:"entity"`
	OrganizationIntegration ConnectionOrganizationIntegration `json:"organizationIntegration"`
	Config                  T                                 `json:"config"`
	Status                  string                            `json:"status"`
	Organization            OrganizationReference             `json:"organization"`
	Project                 ProjectReference                  `json:"project"`
}
