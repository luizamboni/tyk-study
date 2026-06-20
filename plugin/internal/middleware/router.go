package middleware

import (
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"example.com/tky-study/integration-router-plugin/internal/catalog"
	"example.com/tky-study/integration-router-plugin/internal/response"
	"example.com/tky-study/integration-router-plugin/internal/token"
	"github.com/TykTechnologies/tyk/ctx"
	"github.com/TykTechnologies/tyk/log"
)

type Router struct {
	catalog  *catalog.Catalog
	resolver *token.Resolver
	client   *http.Client
}

func NewRouter() *Router {
	return &Router{
		catalog:  catalog.New(),
		resolver: token.NewResolver(token.NewBroker()),
		client:   &http.Client{Timeout: 5 * time.Second},
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

	switch integration.Auth.Type {
	case "oauth":
		if err := router.applyOAuth(request, tenant, integration); err != nil {
			log.Get().WithError(err).Error("integration plugin failed to resolve oauth")
			response.Error(writer, http.StatusBadGateway, "auth_failed", err.Error())
			return
		}
		prepareForProxy(request, tenant, integrationName, operationName, operation)
	case "api-key", "basic":
		upstreamRequest, err := router.buildUpstreamRequest(request, integration, operation)
		if err != nil {
			response.Error(writer, http.StatusBadGateway, "upstream_request_failed", err.Error())
			return
		}
		router.proxyUpstream(writer, upstreamRequest)
	default:
		response.Error(writer, http.StatusInternalServerError, "unknown_auth", integration.Auth.Type)
	}
}

func (router *Router) applyOAuth(request *http.Request, tenant string, integration catalog.Integration) error {
	result, err := router.resolver.Resolve(tenant, integration.Auth.CredentialService)
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+result.Value.AccessToken)
	request.Header.Set("X-Plugin-Token-Source", result.PluginSource)
	request.Header.Set("X-Broker-Token-Source", result.BrokerSource)
	return nil
}

func prepareForProxy(request *http.Request, tenant, integrationName, operationName string, operation catalog.Operation) {
	request.URL.Path = operation.Path
	request.URL.RawPath = ""
	request.Header.Set("X-Tenant-ID", tenant)
	request.Header.Set("X-Integration-Name", integrationName)
	request.Header.Set("X-Integration-Operation", operationName)
}

func (router *Router) buildUpstreamRequest(request *http.Request, integration catalog.Integration, operation catalog.Operation) (*http.Request, error) {
	target, _ := url.Parse(integration.TargetURL)
	target.Path = operation.Path
	target.RawQuery = request.URL.RawQuery

	body, err := io.ReadAll(request.Body)
	if err != nil {
		return nil, err
	}
	defer request.Body.Close()

	upstream, err := http.NewRequest(request.Method, target.String(), strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	upstream.Header.Set("X-Tenant-ID", request.Header.Get("X-Tenant-ID"))

	switch integration.Auth.Type {
	case "api-key":
		key := os.Getenv(integration.Auth.APIKeyEnvVar)
		if key == "" {
			key = "default-demo-key"
		}
		upstream.Header.Set(integration.Auth.APIKeyHeader, key)
	case "basic":
		user := os.Getenv(integration.Auth.BasicUserEnvVar)
		pass := os.Getenv(integration.Auth.BasicPassEnvVar)
		if user == "" {
			user = "demo"
		}
		if pass == "" {
			pass = "demo"
		}
		upstream.SetBasicAuth(user, pass)
	}

	return upstream, nil
}

func (router *Router) proxyUpstream(writer http.ResponseWriter, upstream *http.Request) {
	resp, err := router.client.Do(upstream)
	if err != nil {
		log.Get().WithError(err).Error("integration plugin upstream call failed")
		response.Error(writer, http.StatusBadGateway, "upstream_error", err.Error())
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Get().WithError(err).Error("integration plugin failed to read upstream body")
		response.Error(writer, http.StatusBadGateway, "upstream_read_error", err.Error())
		return
	}

	for key, values := range resp.Header {
		for _, value := range values {
			writer.Header().Add(key, value)
		}
	}
	writer.WriteHeader(resp.StatusCode)
	writer.Write(body)
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
	if len(parts) < 5 {
		return "", "", false
	}
	if parts[0] != "api" || parts[1] != "integration" || parts[3] != "operation" {
		return "", "", false
	}
	return parts[2], parts[4], true
}
