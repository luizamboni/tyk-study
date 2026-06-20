package main

import (
	"net/http"

	"example.com/tky-study/integration-router-plugin/internal/middleware"
	"github.com/TykTechnologies/tyk/log"
)

var router = middleware.NewRouter()

// RouteIntegration is the symbol loaded by Tyk after key authentication.
func RouteIntegration(response http.ResponseWriter, request *http.Request) {
	router.Handle(response, request)
}

func main() {}

func init() {
	log.Get().Info("integration router Go plugin loaded")
}
