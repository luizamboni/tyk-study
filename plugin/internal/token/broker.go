package token

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

type brokerResponse struct {
	AccessToken string    `json:"access_token"`
	ExpiresAt   time.Time `json:"expires_at"`
	Source      string    `json:"source"`
}

type Broker struct {
	baseURL       string
	internalToken string
	client        *http.Client
}

func NewBroker() *Broker {
	baseURL := os.Getenv("TOKEN_BROKER_URL")
	if baseURL == "" {
		baseURL = "http://token-broker:3000"
	}

	return &Broker{
		baseURL:       baseURL,
		internalToken: os.Getenv("TOKEN_BROKER_INTERNAL_TOKEN"),
		client:        &http.Client{Timeout: 2 * time.Second},
	}
}

func (broker *Broker) Resolve(tenant, service string) (Value, string, error) {
	body := strings.NewReader(fmt.Sprintf(`{"tenant_id":%q,"service":%q}`, tenant, service))
	request, err := http.NewRequest(
		http.MethodPost,
		broker.baseURL+"/internal/v1/tokens/resolve",
		body,
	)
	if err != nil {
		return Value{}, "", err
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+broker.internalToken)

	response, err := broker.client.Do(request)
	if err != nil {
		return Value{}, "", fmt.Errorf("token broker indisponível: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return Value{}, "", fmt.Errorf("token broker retornou HTTP %d", response.StatusCode)
	}

	var result brokerResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return Value{}, "", fmt.Errorf("resposta inválida do token broker: %w", err)
	}
	if result.AccessToken == "" || result.ExpiresAt.IsZero() {
		return Value{}, "", fmt.Errorf("token broker retornou token incompleto")
	}

	return Value{
		AccessToken: result.AccessToken,
		ExpiresAt:   result.ExpiresAt,
	}, result.Source, nil
}
