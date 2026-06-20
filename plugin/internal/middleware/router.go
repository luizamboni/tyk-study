package middleware

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"example.com/tky-study/integration-router-plugin/internal/response"
	"example.com/tky-study/integration-router-plugin/internal/token"
	"github.com/TykTechnologies/tyk/ctx"
	"github.com/TykTechnologies/tyk/log"
)

type Router struct {
	resolver *token.Resolver
	client   *http.Client
}

func NewRouter() *Router {
	return &Router{
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

	cred, err := router.resolver.Resolve(tenant, integrationName, operationName, request.Method)
	if err != nil {
		log.Get().WithError(err).Error("credential broker resolve failed")
		response.Error(writer, http.StatusBadGateway, "credential_error", err.Error())
		return
	}

	request.Header.Set("X-Tenant-ID", tenant)
	request.Header.Set("X-Integration-Name", integrationName)
	request.Header.Set("X-Integration-Operation", operationName)

	switch cred.AuthType {
	case "bearer":
		request.Header.Set("Authorization", "Bearer "+cred.AccessToken)
		request.Header.Set("X-Broker-Source", cred.Source)
		request.URL.Path = cred.OperationPath
		request.URL.RawPath = ""

	case "api-key":
		upstream, err := router.buildUpstreamRequest(request, cred)
		if err != nil {
			response.Error(writer, http.StatusBadGateway, "upstream_error", err.Error())
			return
		}
		upstream.Header.Set(cred.HeaderName, cred.HeaderValue)
		router.proxyUpstream(writer, upstream)

	case "basic":
		upstream, err := router.buildUpstreamRequest(request, cred)
		if err != nil {
			response.Error(writer, http.StatusBadGateway, "upstream_error", err.Error())
			return
		}
		upstream.SetBasicAuth(cred.Username, cred.Password)
		router.proxyUpstream(writer, upstream)

	default:
		response.Error(writer, http.StatusInternalServerError, "unknown_auth", cred.AuthType)
	}
}

func (router *Router) buildUpstreamRequest(original *http.Request, cred token.Credential) (*http.Request, error) {
	target, _ := url.Parse(cred.TargetURL)
	target.Path = cred.OperationPath
	target.RawQuery = original.URL.RawQuery

	body, err := io.ReadAll(original.Body)
	if err != nil {
		return nil, err
	}
	defer original.Body.Close()

	upstream, err := http.NewRequest(original.Method, target.String(), strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	upstream.Header.Set("X-Tenant-ID", original.Header.Get("X-Tenant-ID"))
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
