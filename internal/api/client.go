package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/mitchgrogg/rita-devtools-tui/internal/types"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

func New(baseURL string) *Client {
	return &Client{
		BaseURL:    baseURL,
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *Client) GetConfig() (*types.Config, error) {
	var cfg types.Config
	if err := c.doJSON("GET", "/api/config", nil, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Client) PutConfig(cfg types.Config) error {
	return c.doJSON("PUT", "/api/config", cfg, nil)
}

func (c *Client) GetDelays() (*types.Delays, error) {
	var d types.Delays
	if err := c.doJSON("GET", "/api/delays", nil, &d); err != nil {
		return nil, err
	}
	return &d, nil
}

func (c *Client) SetGlobalDelay(ms int) error {
	body := map[string]int{"delay_ms": ms}
	return c.doJSON("PUT", "/api/delays/global", body, nil)
}

func (c *Client) RemoveGlobalDelay() error {
	return c.doJSON("DELETE", "/api/delays/global", nil, nil)
}

func (c *Client) ListPatternDelays() ([]types.PatternDelay, error) {
	var patterns []types.PatternDelay
	if err := c.doJSON("GET", "/api/delays/patterns", nil, &patterns); err != nil {
		return nil, err
	}
	return patterns, nil
}

func (c *Client) AddPatternDelay(p types.PatternDelay) error {
	return c.doJSON("POST", "/api/delays/patterns", p, nil)
}

func (c *Client) RemoveAllPatternDelays() error {
	return c.doJSON("DELETE", "/api/delays/patterns", nil, nil)
}

func (c *Client) RemovePatternDelay(index int) error {
	return c.doJSON("DELETE", fmt.Sprintf("/api/delays/patterns/%d", index), nil, nil)
}

func (c *Client) ListAlterations() ([]types.Alteration, error) {
	var alts []types.Alteration
	if err := c.doJSON("GET", "/api/alterations", nil, &alts); err != nil {
		return nil, err
	}
	return alts, nil
}

func (c *Client) AddAlteration(a types.Alteration) error {
	return c.doJSON("POST", "/api/alterations", a, nil)
}

func (c *Client) RemoveAllAlterations() error {
	return c.doJSON("DELETE", "/api/alterations", nil, nil)
}

func (c *Client) RemoveAlteration(index int) error {
	return c.doJSON("DELETE", fmt.Sprintf("/api/alterations/%d", index), nil, nil)
}

func (c *Client) doJSON(method, path string, reqBody any, respBody any) error {
	var bodyReader io.Reader
	if reqBody != nil {
		data, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.BaseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	if respBody != nil {
		if err := json.NewDecoder(resp.Body).Decode(respBody); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}

	return nil
}
