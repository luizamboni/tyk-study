package token

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

type Credential struct {
	TargetURL     string
	OperationPath string
	AuthType      string
	Source        string

	HeaderName  string
	HeaderValue string

	Username string
	Password string

	AccessToken string
	ExpiresAt   time.Time
}

type brokerEnvelope struct {
	TargetURL     string          `json:"target_url"`
	OperationPath string          `json:"operation_path"`
	AuthType      string          `json:"auth_type"`
	Credentials   json.RawMessage `json:"credentials"`
	ExpiresAt     *time.Time      `json:"expires_at"`
	Source        string          `json:"source"`
}

type apiKeyCreds struct {
	HeaderName  string `json:"header_name"`
	HeaderValue string `json:"header_value"`
}

type basicCreds struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type bearerCreds struct {
	AccessToken string `json:"access_token"`
}

type Broker struct {
	baseURL string
	client  *http.Client
}

func NewBroker() *Broker {
	baseURL := os.Getenv("CREDENTIAL_BROKER_URL")
	if baseURL == "" {
		baseURL = "http://token-broker:3000"
	}

	return &Broker{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 5 * time.Second},
	}
}

func (broker *Broker) Resolve(tenant, integration, operation, method string) (Credential, error) {
	body := strings.NewReader(fmt.Sprintf(
		`{"tenant_id":%q,"integration":%q,"operation":%q,"method":%q}`,
		tenant, integration, operation, method,
	))
	req, err := http.NewRequest(
		http.MethodPost,
		broker.baseURL+"/internal/v1/credentials/resolve",
		body,
	)
	if err != nil {
		return Credential{}, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := broker.client.Do(req)
	if err != nil {
		return Credential{}, fmt.Errorf("credential broker indisponível: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Credential{}, fmt.Errorf("credential broker retornou HTTP %d", resp.StatusCode)
	}

	var envelope brokerEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return Credential{}, fmt.Errorf("resposta inválida do broker: %w", err)
	}

	cred := Credential{
		TargetURL:     envelope.TargetURL,
		OperationPath: envelope.OperationPath,
		AuthType:      envelope.AuthType,
		Source:        envelope.Source,
	}

	if envelope.ExpiresAt != nil {
		cred.ExpiresAt = *envelope.ExpiresAt
	}

	switch envelope.AuthType {
	case "api-key":
		var c apiKeyCreds
		if err := json.Unmarshal(envelope.Credentials, &c); err != nil {
			return Credential{}, fmt.Errorf("credenciais api-key inválidas: %w", err)
		}
		cred.HeaderName = c.HeaderName
		cred.HeaderValue = c.HeaderValue

	case "basic":
		var c basicCreds
		if err := json.Unmarshal(envelope.Credentials, &c); err != nil {
			return Credential{}, fmt.Errorf("credenciais basic inválidas: %w", err)
		}
		cred.Username = c.Username
		cred.Password = c.Password

	case "bearer":
		var c bearerCreds
		if err := json.Unmarshal(envelope.Credentials, &c); err != nil {
			return Credential{}, fmt.Errorf("credenciais bearer inválidas: %w", err)
		}
		cred.AccessToken = c.AccessToken

	default:
		return Credential{}, fmt.Errorf("tipo de auth desconhecido: %s", envelope.AuthType)
	}

	return cred, nil
}
