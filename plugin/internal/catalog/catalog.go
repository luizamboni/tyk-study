package catalog

import "net/http"

type Operation struct {
	Path              string
	CredentialService string
}

type Catalog struct {
	operations map[string]operation
}

type operation struct {
	method            string
	path              string
	credentialService string
}

func New() *Catalog {
	return &Catalog{
		operations: map[string]operation{
			"oauth/resource": {
				method:            http.MethodGet,
				path:              "/resource",
				credentialService: "oauth-demo",
			},
		},
	}
}

func (catalog *Catalog) Find(service, action, method string) (Operation, bool) {
	operation, found := catalog.operations[service+"/"+action]
	if !found || operation.method != method {
		return Operation{}, false
	}

	return Operation{
		Path:              operation.path,
		CredentialService: operation.credentialService,
	}, true
}
