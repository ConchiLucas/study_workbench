package imagegen

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/conchi/study-content-admin/internal/configclient"
)

const (
	defaultSize    = "1024x1024"
	fallbackSize   = "1536x864"
	defaultQuality = "high"
	maxBodyBytes   = 25 << 20
)

type Client struct {
	http *http.Client
}

func New() *Client {
	return &Client{http: &http.Client{Timeout: 180 * time.Second}}
}

// GeneratePNG calls an openai-compatible /v1/images/generations endpoint.
func (c *Client) GeneratePNG(ctx context.Context, provider configclient.AIProvider, prompt, negative string) ([]byte, error) {
	if provider.BaseURL == "" || provider.Model == "" {
		return nil, fmt.Errorf("图片模型未配置 baseUrl/model")
	}
	endpoint, err := generationsURL(provider.BaseURL)
	if err != nil {
		return nil, err
	}

	size := option(provider.Options, "size", defaultSize)
	quality := option(provider.Options, "quality", defaultQuality)
	responseFormat := option(provider.Options, "responseFormat", "b64_json")
	if responseFormat != "b64_json" {
		return nil, fmt.Errorf("仅支持 b64_json 返回格式")
	}

	png, err := c.postGenerate(ctx, provider, endpoint, prompt, negative, size, quality)
	if err != nil && size == defaultSize {
		// Some gateways only advertise landscape sizes.
		return c.postGenerate(ctx, provider, endpoint, prompt, negative, fallbackSize, quality)
	}
	return png, err
}

func (c *Client) postGenerate(
	ctx context.Context,
	provider configclient.AIProvider,
	endpoint, prompt, negative, size, quality string,
) ([]byte, error) {
	body := map[string]any{
		"model":           provider.Model,
		"prompt":          prompt,
		"response_format": "b64_json",
	}
	if isGrokImagine(provider.Model) {
		if negative != "" {
			body["prompt"] = prompt + "\n\nAvoid: " + negative
		}
		body["aspect_ratio"] = aspectRatio(size)
	} else {
		if negative != "" {
			body["negative_prompt"] = negative
		}
		body["quality"] = quality
		body["size"] = size
	}

	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if provider.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+provider.APIKey)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("调用图片模型失败: %w", err)
	}
	defer res.Body.Close()
	limited := io.LimitReader(res.Body, maxBodyBytes+1)
	respBody, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if len(respBody) > maxBodyBytes {
		return nil, fmt.Errorf("图片生成响应过大")
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		msg := strings.TrimSpace(string(respBody))
		if len(msg) > 400 {
			msg = msg[:400]
		}
		return nil, fmt.Errorf("图片模型 HTTP %d: %s", res.StatusCode, msg)
	}

	var parsed struct {
		Data []struct {
			B64JSON string `json:"b64_json"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("解析图片响应失败: %w", err)
	}
	if len(parsed.Data) == 0 || parsed.Data[0].B64JSON == "" {
		return nil, fmt.Errorf("图片生成返回为空")
	}
	png, err := base64.StdEncoding.DecodeString(parsed.Data[0].B64JSON)
	if err != nil {
		return nil, fmt.Errorf("图片 base64 无效: %w", err)
	}
	if len(png) == 0 {
		return nil, fmt.Errorf("图片数据为空")
	}
	return png, nil
}

// PickProvider chooses active image provider, else first enabled IMAGE_GENERATION provider.
func PickProvider(ai configclient.AIConfiguration) (configclient.AIProvider, error) {
	byID := map[string]configclient.AIProvider{}
	var first *configclient.AIProvider
	for i := range ai.Providers {
		p := ai.Providers[i]
		if !executableImage(p) {
			continue
		}
		byID[p.ID] = p
		if first == nil {
			cp := p
			first = &cp
		}
	}
	if first == nil {
		return configclient.AIProvider{}, fmt.Errorf("配置中心没有可用的图片模型（需 openai-compatible + IMAGE_GENERATION）")
	}
	if p, ok := byID[ai.ActiveProviderID]; ok {
		return p, nil
	}
	return *first, nil
}

func executableImage(p configclient.AIProvider) bool {
	if !p.Enabled || !strings.EqualFold(p.Type, "openai-compatible") {
		return false
	}
	if strings.TrimSpace(p.ID) == "" || strings.TrimSpace(p.BaseURL) == "" || strings.TrimSpace(p.Model) == "" {
		return false
	}
	hasGen := false
	for _, cap := range p.Capabilities {
		if cap == "IMAGE_GENERATION" {
			hasGen = true
			break
		}
	}
	return hasGen
}

func isGrokImagine(model string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(model)), "grok-imagine-image")
}

func aspectRatio(size string) string {
	parts := strings.Split(strings.ToLower(size), "x")
	if len(parts) != 2 {
		return "1:1"
	}
	var w, h int
	if _, err := fmt.Sscanf(parts[0], "%d", &w); err != nil || w <= 0 {
		return "1:1"
	}
	if _, err := fmt.Sscanf(parts[1], "%d", &h); err != nil || h <= 0 {
		return "1:1"
	}
	g := gcd(w, h)
	return fmt.Sprintf("%d:%d", w/g, h/g)
}

func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	if a == 0 {
		return 1
	}
	return a
}

func option(opts map[string]string, key, def string) string {
	if opts == nil {
		return def
	}
	if v := strings.TrimSpace(opts[key]); v != "" {
		return v
	}
	return def
}

func generationsURL(base string) (string, error) {
	base = strings.TrimSpace(base)
	u, err := url.Parse(base)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", fmt.Errorf("图片生成服务地址无效")
	}
	p := strings.TrimRight(u.Path, "/")
	lower := strings.ToLower(p)
	if strings.HasSuffix(lower, "/images/generations") || strings.HasSuffix(lower, "/images/edits") {
		return "", fmt.Errorf("图片生成服务地址不能包含 images 接口")
	}
	if !strings.HasSuffix(lower, "/v1") {
		p += "/v1"
	}
	p += "/images/generations"
	u.Path = p
	u.RawQuery = ""
	u.Fragment = ""
	return u.String(), nil
}
