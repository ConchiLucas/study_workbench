package tts

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/conchi/study-content-admin/internal/configclient"
)

type Client struct {
	http *http.Client
}

func New() *Client {
	return &Client{http: &http.Client{Timeout: 60 * time.Second}}
}

type grokRequestBody struct {
	Text     string `json:"text"`
	VoiceID  string `json:"voice_id"`
	Language string `json:"language"`
}

type mimoMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type mimoAudioParams struct {
	Format string `json:"format"`
	Voice  string `json:"voice,omitempty"`
}

type mimoRequestBody struct {
	Model    string           `json:"model"`
	Messages []mimoMessage    `json:"messages"`
	Audio    mimoAudioParams  `json:"audio"`
	Stream   bool             `json:"stream"`
}

type mimoResponse struct {
	Choices []struct {
		Message struct {
			Audio *struct {
				Data string `json:"data"`
			} `json:"audio"`
		} `json:"message"`
	} `json:"choices"`
}

// Synthesize calls the provider TTS API and returns audio bytes (prefer MPEG).
func (c *Client) Synthesize(ctx context.Context, provider configclient.AIProvider, text string) ([]byte, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("朗读文本为空")
	}
	base := strings.TrimRight(strings.TrimSpace(provider.BaseURL), "/")
	if base == "" {
		return nil, fmt.Errorf("TTS Provider 缺少 baseUrl")
	}
	if strings.TrimSpace(provider.APIKey) == "" {
		return nil, fmt.Errorf("TTS Provider 缺少 apiKey")
	}
	if isMimoProvider(provider) {
		return c.synthesizeMimo(ctx, provider, base, text)
	}
	return c.synthesizeGrok(ctx, provider, base, text)
}

func isMimoProvider(provider configclient.AIProvider) bool {
	typeName := strings.ToLower(strings.TrimSpace(provider.Type))
	id := strings.ToLower(strings.TrimSpace(provider.ID))
	model := strings.ToLower(strings.TrimSpace(provider.Model))
	if typeName == "mimo-tts" || strings.Contains(typeName, "mimo") {
		return true
	}
	if strings.Contains(id, "xiaomi") || strings.Contains(id, "mimo") {
		return true
	}
	return strings.Contains(model, "mimo") && strings.Contains(model, "tts")
}

func (c *Client) synthesizeGrok(ctx context.Context, provider configclient.AIProvider, base, text string) ([]byte, error) {
	voice := strings.TrimSpace(provider.Voice)
	if voice == "" {
		voice = "eve"
	}
	language := "zh"
	if provider.Options != nil {
		if v := strings.TrimSpace(provider.Options["language"]); v != "" && !strings.EqualFold(v, "auto") {
			language = v
		}
	}
	payload, err := json.Marshal(grokRequestBody{Text: text, VoiceID: voice, Language: language})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/tts", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+provider.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "audio/mpeg, application/octet-stream, */*")
	return c.doAudio(req)
}

func (c *Client) synthesizeMimo(ctx context.Context, provider configclient.AIProvider, base, text string) ([]byte, error) {
	model := strings.TrimSpace(provider.Model)
	if model == "" {
		model = "mimo-v2.5-tts"
	}
	voice := strings.TrimSpace(provider.Voice)
	if voice == "" {
		voice = "mimo_default"
	}
	format := "mp3"
	if provider.Options != nil {
		if v := strings.TrimSpace(provider.Options["format"]); v != "" {
			format = strings.ToLower(v)
		}
		if v := strings.TrimSpace(provider.Options["voice"]); v != "" {
			voice = v
		}
	}
	style := "用清晰、平稳、适合幼儿识字课的语速朗读单个汉字。"
	if provider.Options != nil {
		if v := strings.TrimSpace(provider.Options["style"]); v != "" {
			style = v
		}
	}
	body := mimoRequestBody{
		Model: model,
		Messages: []mimoMessage{
			{Role: "user", Content: style},
			{Role: "assistant", Content: text},
		},
		Audio:  mimoAudioParams{Format: format, Voice: voice},
		Stream: false,
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+provider.APIKey)
	req.Header.Set("api-key", provider.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("调用 TTS 失败: %w", err)
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 12<<20))
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		msg := strings.TrimSpace(string(raw))
		if len(msg) > 300 {
			msg = msg[:300]
		}
		return nil, fmt.Errorf("TTS 上游返回 %d: %s", res.StatusCode, msg)
	}
	var parsed mimoResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("解析 MiMo TTS 响应失败: %w", err)
	}
	if len(parsed.Choices) == 0 || parsed.Choices[0].Message.Audio == nil {
		return nil, fmt.Errorf("MiMo TTS 未返回音频")
	}
	b64 := strings.TrimSpace(parsed.Choices[0].Message.Audio.Data)
	if b64 == "" {
		return nil, fmt.Errorf("MiMo TTS 音频为空")
	}
	audio, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, fmt.Errorf("解码 MiMo TTS 音频失败: %w", err)
	}
	if len(audio) == 0 {
		return nil, fmt.Errorf("TTS 上游返回空音频")
	}
	return audio, nil
}

func (c *Client) doAudio(req *http.Request) ([]byte, error) {
	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("调用 TTS 失败: %w", err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		msg := strings.TrimSpace(string(body))
		if len(msg) > 300 {
			msg = msg[:300]
		}
		return nil, fmt.Errorf("TTS 上游返回 %d: %s", res.StatusCode, msg)
	}
	if len(body) == 0 {
		return nil, fmt.Errorf("TTS 上游返回空音频")
	}
	return body, nil
}

func enabledTTSProviders(ai configclient.AIConfiguration) []configclient.AIProvider {
	var enabled []configclient.AIProvider
	for _, p := range ai.Providers {
		if !p.Enabled || strings.TrimSpace(p.BaseURL) == "" {
			continue
		}
		hasTTS := false
		for _, c := range p.Capabilities {
			if strings.EqualFold(strings.TrimSpace(c), "AUDIO_TTS") {
				hasTTS = true
				break
			}
		}
		if !hasTTS {
			continue
		}
		enabled = append(enabled, p)
	}
	return enabled
}

// WithLanguage returns a shallow copy of provider with Options["language"] set.
func WithLanguage(provider configclient.AIProvider, language string) configclient.AIProvider {
	language = strings.TrimSpace(language)
	if language == "" {
		return provider
	}
	out := provider
	opts := make(map[string]string, len(provider.Options)+1)
	for k, v := range provider.Options {
		opts[k] = v
	}
	opts["language"] = language
	out.Options = opts
	return out
}

// PickProvider prefers voice-models activeProviderId, then grok-tts, then first AUDIO_TTS.
func PickProvider(ai configclient.AIConfiguration) (configclient.AIProvider, error) {
	enabled := enabledTTSProviders(ai)
	if len(enabled) == 0 {
		return configclient.AIProvider{}, fmt.Errorf("配置中心没有可用的 TTS Provider（需 AUDIO_TTS）")
	}
	for _, p := range enabled {
		if p.ID == ai.ActiveProviderID {
			return p, nil
		}
	}
	for _, p := range enabled {
		typeName := strings.ToLower(strings.TrimSpace(p.Type))
		id := strings.ToLower(strings.TrimSpace(p.ID))
		if typeName == "grok-tts" || (strings.Contains(id, "grok") && strings.Contains(id, "tts")) {
			return p, nil
		}
	}
	return enabled[0], nil
}

// PickProviderByID selects an enabled AUDIO_TTS provider by id (exact), else fuzzy match.
func PickProviderByID(ai configclient.AIConfiguration, providerID string) (configclient.AIProvider, error) {
	providerID = strings.TrimSpace(providerID)
	if providerID == "" {
		return PickProvider(ai)
	}
	enabled := enabledTTSProviders(ai)
	if len(enabled) == 0 {
		return configclient.AIProvider{}, fmt.Errorf("配置中心没有可用的 TTS Provider（需 AUDIO_TTS）")
	}
	for _, p := range enabled {
		if p.ID == providerID {
			return p, nil
		}
	}
	want := strings.ToLower(providerID)
	for _, p := range enabled {
		id := strings.ToLower(strings.TrimSpace(p.ID))
		typeName := strings.ToLower(strings.TrimSpace(p.Type))
		if id == want || strings.Contains(id, want) || typeName == want {
			return p, nil
		}
	}
	return configclient.AIProvider{}, fmt.Errorf("未找到 TTS Provider: %s", providerID)
}
