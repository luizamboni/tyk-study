package middleware

import (
	"net/http"
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
		response.Error(writer, http.StatusForbidden, "missing_tenant", "tenant_id não associado à chave Tyk")
		return
	}

	service, action, valid := routeFromPath(request.URL.Path)
	if !valid {
		response.Error(writer, http.StatusNotFound, "unknown_operation", "use /integrations/{service}/{action}")
		return
	}

	operation, found := router.catalog.Find(service, action, request.Method)
	if !found {
		response.Error(writer, http.StatusForbidden, "operation_not_allowed", service+"/"+action)
		return
	}

	result, err := router.resolver.Resolve(tenant, operation.CredentialService)
	if err != nil {
		log.Get().WithError(err).Error("integration plugin failed to resolve token")
		response.Error(writer, http.StatusBadGateway, "credential_unavailable", err.Error())
		return
	}

	prepareUpstreamRequest(request, tenant, service, action, operation, result)
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
	if len(parts) < 3 || parts[0] != "integrations" {
		return "", "", false
	}

	return parts[1], parts[2], true
}

func prepareUpstreamRequest(
	request *http.Request,
	tenant string,
	service string,
	action string,
	operation catalog.Operation,
	result token.Result,
) {
	request.Header.Del("Authorization")
	request.Header.Set("Authorization", "Bearer "+result.Value.AccessToken)
	request.Header.Set("X-Tenant-ID", tenant)
	request.Header.Set("X-Integration-Service", service)
	request.Header.Set("X-Integration-Action", action)
	request.Header.Set("X-Plugin-Token-Source", result.PluginSource)
	request.Header.Set("X-Broker-Token-Source", result.BrokerSource)
	request.URL.Path = operation.Path
	request.URL.RawPath = ""
}
