package kv

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// UpstashConfig holds the REST API credentials for Upstash Redis.
// Set via KV_REST_API_URL + KV_REST_API_TOKEN, or auto-parsed from REDIS_URL.
type UpstashConfig struct {
	URL   string
	Token string
}

var (
	cachedCfg   *UpstashConfig
	cachedHTTP   *http.Client
)

func getHTTPClient() *http.Client {
	if cachedHTTP == nil {
		cachedHTTP = &http.Client{Timeout: 10 * time.Second}
	}
	return cachedHTTP
}

// LoadConfig reads Upstash credentials from environment.
// Supports KV_REST_API_URL+KV_REST_API_TOKEN or REDIS_URL (rediss://...).
func LoadConfig() *UpstashConfig {
	if cachedCfg != nil {
		return cachedCfg
	}

	restURL := strings.TrimSpace(os.Getenv("KV_REST_API_URL"))
	restToken := strings.TrimSpace(os.Getenv("KV_REST_API_TOKEN"))
	if restURL != "" && restToken != "" {
		cachedCfg = &UpstashConfig{URL: restURL, Token: restToken}
		return cachedCfg
	}

	// Try parsing REDIS_URL (rediss://default:<token>@<host>:<port>)
	redisURL := strings.TrimSpace(os.Getenv("REDIS_URL"))
	if redisURL != "" {
		cfg, err := parseRedisURL(redisURL)
		if err == nil {
			cachedCfg = cfg
			return cachedCfg
		}
	}

	return nil
}

func parseRedisURL(raw string) (*UpstashConfig, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	host := u.Hostname()
	if host == "" {
		return nil, fmt.Errorf("no host in REDIS_URL")
	}
	token := ""
	if u.User != nil {
		token = u.User.Username()
	}
	return &UpstashConfig{
		URL:   "https://" + host,
		Token: token,
	}, nil
}

// Enabled returns true if Upstash is configured.
func Enabled() bool {
	return LoadConfig() != nil
}

// Get retrieves a value by key. Returns ("", nil) if key does not exist.
func Get(ctx context.Context, key string) (string, error) {
	cfg := LoadConfig()
	if cfg == nil {
		return "", fmt.Errorf("kv: not configured")
	}

	body, _ := json.Marshal([]string{key})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.URL+"/v2/get", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := getHTTPClient().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)

	var result struct {
		Result json.RawMessage `json:"result"`
		Error  string          `json:"error"`
	}
	if err := json.Unmarshal(b, &result); err != nil {
		return "", fmt.Errorf("kv: unmarshal response: %w", err)
	}
	if result.Error != "" {
		return "", fmt.Errorf("kv: %s", result.Error)
	}
	if string(result.Result) == "null" {
		return "", nil
	}

	var val string
	if err := json.Unmarshal(result.Result, &val); err != nil {
		return "", nil
	}
	return val, nil
}

// Set stores a key-value pair.
func Set(ctx context.Context, key, value string) error {
	cfg := LoadConfig()
	if cfg == nil {
		return fmt.Errorf("kv: not configured")
	}

	payload := []any{key, value}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.URL+"/v2/set", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := getHTTPClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)

	var result struct {
		Result string `json:"result"`
		Error  string `json:"error"`
	}
	if err := json.Unmarshal(b, &result); err != nil {
		return fmt.Errorf("kv: unmarshal response: %w", err)
	}
	if result.Error != "" {
		return fmt.Errorf("kv: %s", result.Error)
	}
	return nil
}
