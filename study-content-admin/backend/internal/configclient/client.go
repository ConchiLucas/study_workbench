package configclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

var ErrUnavailable = errors.New("无法加载共享配置中心")
var ErrNotLoaded = errors.New("配置尚未加载")

type Client struct {
	baseURL string
	http    *http.Client
	mu      sync.RWMutex
	current RuntimeConfiguration
	loaded  bool
}

func New(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		http:    &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *Client) BaseURL() string { return c.baseURL }

func (c *Client) Load(ctx context.Context) (RuntimeConfiguration, error) {
	return c.Refresh(ctx)
}

func (c *Client) Refresh(ctx context.Context) (RuntimeConfiguration, error) {
	if c.baseURL == "" {
		return RuntimeConfiguration{}, ErrUnavailable
	}
	value, err := c.getRuntime(ctx)
	if err != nil {
		return RuntimeConfiguration{}, err
	}
	c.mu.Lock()
	c.current, c.loaded = value, true
	c.mu.Unlock()
	return clone(value), nil
}

func (c *Client) Current() (RuntimeConfiguration, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if !c.loaded {
		return RuntimeConfiguration{}, ErrNotLoaded
	}
	return clone(c.current), nil
}

func (c *Client) Require(ctx context.Context) (RuntimeConfiguration, error) {
	if value, err := c.Current(); err == nil {
		return value, nil
	}
	return c.Load(ctx)
}

func (c *Client) LoadImageModels(ctx context.Context) (AIConfiguration, error) {
	return c.getAISection(ctx, "/api/admin/v1/configuration/image-models")
}

func (c *Client) LoadVideoModels(ctx context.Context) (AIConfiguration, error) {
	return c.getAISection(ctx, "/api/admin/v1/configuration/video-models")
}

func (c *Client) LoadVoiceModels(ctx context.Context) (AIConfiguration, error) {
	return c.getAISection(ctx, "/api/admin/v1/configuration/voice-models")
}

func (c *Client) getRuntime(ctx context.Context) (RuntimeConfiguration, error) {
	body, err := c.getJSON(ctx, "/api/runtime/v1/configuration")
	if err != nil {
		return RuntimeConfiguration{}, err
	}
	var value RuntimeConfiguration
	if err := json.Unmarshal(body, &value); err != nil {
		return RuntimeConfiguration{}, ErrUnavailable
	}
	if value.SchemaVersion != "1" {
		return RuntimeConfiguration{}, ErrUnavailable
	}
	return value, nil
}

func (c *Client) getAISection(ctx context.Context, path string) (AIConfiguration, error) {
	body, err := c.getJSON(ctx, path)
	if err != nil {
		return AIConfiguration{}, err
	}
	var value AIConfiguration
	if err := json.Unmarshal(body, &value); err != nil {
		return AIConfiguration{}, ErrUnavailable
	}
	if value.Providers == nil {
		value.Providers = []AIProvider{}
	}
	return value, nil
}

func (c *Client) getJSON(ctx context.Context, path string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, ErrUnavailable
	}
	req.Header.Set("Accept", "application/json")
	res, err := c.http.Do(req)
	if err != nil {
		return nil, ErrUnavailable
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %s", ErrUnavailable, res.Status)
	}
	var raw json.RawMessage
	if err := json.NewDecoder(res.Body).Decode(&raw); err != nil {
		return nil, ErrUnavailable
	}
	return raw, nil
}

func clone(value RuntimeConfiguration) RuntimeConfiguration {
	raw, err := json.Marshal(value)
	if err != nil {
		return value
	}
	var copied RuntimeConfiguration
	_ = json.Unmarshal(raw, &copied)
	return copied
}
