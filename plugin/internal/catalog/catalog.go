package catalog

type AuthConfig struct {
	Type string

	APIKeyHeader string
	APIKeyEnvVar string

	BasicUserEnvVar string
	BasicPassEnvVar string

	CredentialService string
}

type Integration struct {
	TargetURL string
	Auth      AuthConfig
}

type Operation struct {
	Integration string
	Path        string
	Method      string
}

type Catalog struct {
	integrations map[string]Integration
	operations   map[string]Operation
}

func New() *Catalog {
	return &Catalog{
		integrations: map[string]Integration{
			"httpbin": {
				TargetURL: "http://upstream",
				Auth: AuthConfig{
					Type:         "api-key",
					APIKeyHeader: "X-Api-Key",
					APIKeyEnvVar: "INTEGRATION_HTTPBIN_KEY",
				},
			},
			"oauth": {
				TargetURL: "http://protected-api:3000",
				Auth: AuthConfig{
					Type:              "oauth",
					CredentialService: "oauth-demo",
				},
			},
			"echo": {
				TargetURL: "http://upstream",
				Auth: AuthConfig{
					Type:            "basic",
					BasicUserEnvVar: "INTEGRATION_ECHO_USER",
					BasicPassEnvVar: "INTEGRATION_ECHO_PASS",
				},
			},
		},
		operations: map[string]Operation{
			"httpbin:get": {
				Integration: "httpbin",
				Path:        "/get",
				Method:      "GET",
			},
			"httpbin:post": {
				Integration: "httpbin",
				Path:        "/post",
				Method:      "POST",
			},
			"oauth:resource": {
				Integration: "oauth",
				Path:        "/resource",
				Method:      "GET",
			},
			"echo:headers": {
				Integration: "echo",
				Path:        "/headers",
				Method:      "GET",
			},
		},
	}
}

func (c *Catalog) FindOperation(integration, operation, method string) (Operation, bool) {
	op, found := c.operations[integration+":"+operation]
	if !found || op.Method != method {
		return Operation{}, false
	}
	return op, true
}

func (c *Catalog) FindIntegration(name string) (Integration, bool) {
	integration, found := c.integrations[name]
	return integration, found
}
