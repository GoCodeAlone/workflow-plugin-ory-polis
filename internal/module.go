package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type oryPolisModule struct {
	name   string
	config map[string]any
}

func newOryPolisModule(name string, config map[string]any) (*oryPolisModule, error) {
	return &oryPolisModule{name: name, config: config}, nil
}

func (m *oryPolisModule) Init() error {
	baseURL := firstNonEmpty(m.config, "base_url", "baseUrl", "admin_url", "adminUrl", "url")
	if baseURL == "" {
		return fmt.Errorf("ory.polis %q: base_url is required", m.name)
	}
	apiKey := firstNonEmpty(m.config, "api_key", "apiKey", "token")
	client, err := newPolisHTTPClient(baseURL, apiKey)
	if err != nil {
		return fmt.Errorf("ory.polis %q: create client: %w", m.name, err)
	}
	RegisterClient(m.name, &OryPolisClient{HTTP: client, BaseURL: strings.TrimRight(baseURL, "/")})
	return nil
}

func (m *oryPolisModule) Start(context.Context) error { return nil }

func (m *oryPolisModule) Stop(context.Context) error {
	UnregisterClient(m.name)
	return nil
}

type polisHTTPClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func newPolisHTTPClient(rawURL, apiKey string) (*polisHTTPClient, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return nil, err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("url must include scheme and host")
	}
	return &polisHTTPClient{
		baseURL:    strings.TrimRight(parsed.String(), "/"),
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (c *polisHTTPClient) do(ctx context.Context, method, path string, query map[string]string, body any) (any, error) {
	endpoint, err := url.Parse(c.baseURL + path)
	if err != nil {
		return nil, err
	}
	q := endpoint.Query()
	for key, value := range query {
		if value != "" {
			q.Set(key, value)
		}
	}
	endpoint.RawQuery = q.Encode()

	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
		reader = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "workflow-plugin-ory-polis/"+Version)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("polis %s %s: status %d: %s", method, path, resp.StatusCode, strings.TrimSpace(string(data)))
	}
	if len(data) == 0 {
		return map[string]any{}, nil
	}
	var decoded any
	if err := json.Unmarshal(data, &decoded); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return decoded, nil
}
