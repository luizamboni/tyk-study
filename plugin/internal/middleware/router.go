package middleware

import (
	"net/http"
	"net/url"
	"os"
	"strings"

	"example.com/tky-study/integration-router-plugin/internal/catalog"
	"example.com/tky-study/integration-router-plugin/internal/response"
	"example.com/tky-study/integration-router-plugin/internal/token"
	"github.com/TykTechnologies/tyk/ctx"
	"github.com/TykTechnologies/tyk/log"
)

type Router struct {
	catalog  *catalog.Catalog
	resolver *token.Resolver
}

func NewRouter() *Router {
	return &Router{
		catalog:  catalog.New(),
		resolver: token.NewResolver(token.NewBroker()),
	}
}

func (router *Router) Handle(writer http.ResponseWriter, request *http.Request) {
	tenant, valid := tenantFromSession(request)
	if !valid {
		response.Error(writer, http.StatusForbidden, "missing_tenant", "tenant_id nao associado a chave Tyk")
		return
	}

	integrationName, operationName, valid := routeFromPath(request.URL.Path)
	if !valid {
		response.Error(writer, http.StatusNotFound, "invalid_path", "use /api/integration/{name}/operation/{op}")
		return
	}

	operation, found := router.catalog.FindOperation(integrationName, operationName, request.Method)
	if !found {
		response.Error(writer, http.StatusForbidden, "operation_not_allowed", integrationName+"/"+operationName)
		return
	}

	integration, found := router.catalog.FindIntegration(operation.Integration)
	if !found {
		response.Error(writer, http.StatusNotFound, "integration_not_found", operation.Integration)
		return
	}

	if err := router.applyAuth(request, tenant, integration, operation); err != nil {
		log.Get().WithError(err).Error("integration plugin failed to apply auth")
		response.Error(writer, http.StatusBadGateway, "auth_failed", err.Error())
		return
	}

	setTargetURL(request, tenant, integrationName, operationName, integration, operation)
}

func (router *Router) applyAuth(request *http.Request, tenant string, integration catalog.Integration, operation catalog.Operation) error {
	switch integration.Auth.Type {
	case "oauth":
		return router.applyOAuth(request, tenant, integration, operation)
	case "api-key":
		return applyAPIKey(request, integration)
	case "basic":
		return applyBasicAuth(request, integration)
	}
	return nil
}

func (router *Router) applyOAuth(request *http.Request, tenant string, integration catalog.Integration, operation catalog.Operation) error {
	result, err := router.resolver.Resolve(tenant, integration.Auth.CredentialService)
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+result.Value.AccessToken)
	request.Header.Set("X-Plugin-Token-Source", result.PluginSource)
	request.Header.Set("X-Broker-Token-Source", result.BrokerSource)
	return nil
}

func applyAPIKey(request *http.Request, integration catalog.Integration) error {
	key := os.Getenv(integration.Auth.APIKeyEnvVar)
	if key == "" {
		key = "default-demo-key"
	}
	request.Header.Set(integration.Auth.APIKeyHeader, key)
	return nil
}

func applyBasicAuth(request *http.Request, integration catalog.Integration) error {
	user := os.Getenv(integration.Auth.BasicUserEnvVar)
	pass := os.Getenv(integration.Auth.BasicPassEnvVar)
	if user == "" {
		user = "demo"
	}
	if pass == "" {
		pass = "demo"
	}
	request.SetBasicAuth(user, pass)
	return nil
}

func tenantFromSession(request *http.Request) (string, bool) {
	session := ctx.GetSession(request)
	if session == nil {
		return "", false
	}
	tenant, valid := session.MetaData["tenant_id"].(string)
	return tenant, valid && tenant != ""
}

func routeFromPath(path string) (string, string, bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 6 {
		return "", "", false
	}
	if parts[0] != "api" || parts[1] != "integration" || parts[3] != "operation" {
		return "", "", false
	}
	return parts[2], parts[4], true
}

func setTargetURL(request *http.Request, tenant, integrationName, operationName string, integration catalog.Integration, operation catalog.Operation) {
	target, _ := url.Parse(integration.TargetURL)
	target.Path = operation.Path
	request.URL = target
	request.Header.Set("X-Tenant-ID", tenant)
	request.Header.Set("X-Integration-Name", integrationName)
	request.Header.Set("X-Integration-Operation", operationName)
}
