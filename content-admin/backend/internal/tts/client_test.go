package tts_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/conchi/study-content-admin/internal/configclient"
	"github.com/conchi/study-content-admin/internal/tts"
)

func TestPickProviderPrefersActiveThenGrok(t *testing.T) {
	ai := configclient.AIConfiguration{
		ActiveProviderID: "sub2api-grok-tts",
		Providers: []configclient.AIProvider{
			{ID: "xiaomi-mimo-tts", Type: "mimo-tts", BaseURL: "https://mimo", Enabled: true, Capabilities: []string{"AUDIO_TTS"}},
			{ID: "sub2api-grok-tts", Type: "grok-tts", BaseURL: "http://tts", Voice: "eve", Enabled: true, Capabilities: []string{"AUDIO_TTS"}},
		},
	}
	p, err := tts.PickProvider(ai)
	require.NoError(t, err)
	require.Equal(t, "sub2api-grok-tts", p.ID)

	ai.ActiveProviderID = "writer"
	p, err = tts.PickProvider(ai)
	require.NoError(t, err)
	require.Equal(t, "sub2api-grok-tts", p.ID)
}

func TestSynthesizePostsTTSBody(t *testing.T) {
	var gotAuth, gotCT, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/tts", r.URL.Path)
		gotAuth = r.Header.Get("Authorization")
		gotCT = r.Header.Get("Content-Type")
		raw, _ := io.ReadAll(r.Body)
		gotBody = string(raw)
		w.Header().Set("Content-Type", "audio/mpeg")
		_, _ = w.Write([]byte("ID3fake"))
	}))
	defer srv.Close()

	client := tts.New()
	audio, err := client.Synthesize(context.Background(), configclient.AIProvider{
		Type:    "grok-tts",
		BaseURL: srv.URL + "/v1",
		APIKey:  "sk-test",
		Voice:   "eve",
		Options: map[string]string{"language": "auto"},
	}, "一")
	require.NoError(t, err)
	require.Equal(t, []byte("ID3fake"), audio)
	require.Equal(t, "Bearer sk-test", gotAuth)
	require.Equal(t, "application/json", gotCT)
	require.Contains(t, gotBody, `"text":"一"`)
	require.Contains(t, gotBody, `"voice_id":"eve"`)
	require.Contains(t, gotBody, `"language":"zh"`)
}

func TestPickProviderByID(t *testing.T) {
	ai := configclient.AIConfiguration{
		ActiveProviderID: "sub2api-grok-tts",
		Providers: []configclient.AIProvider{
			{ID: "xiaomi-mimo-tts", Type: "mimo-tts", BaseURL: "https://mimo", Enabled: true, Capabilities: []string{"AUDIO_TTS"}},
			{ID: "sub2api-grok-tts", Type: "grok-tts", BaseURL: "http://tts", Voice: "eve", Enabled: true, Capabilities: []string{"AUDIO_TTS"}},
		},
	}
	p, err := tts.PickProviderByID(ai, "xiaomi-mimo-tts")
	require.NoError(t, err)
	require.Equal(t, "xiaomi-mimo-tts", p.ID)
}

func TestSynthesizeMimoUsesChatCompletions(t *testing.T) {
	var gotPath, gotBody, gotAPIKey string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAPIKey = r.Header.Get("api-key")
		raw, _ := io.ReadAll(r.Body)
		gotBody = string(raw)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"audio":{"data":"SUQzZmFrZQ=="}}}]}`))
	}))
	defer srv.Close()

	client := tts.New()
	audio, err := client.Synthesize(context.Background(), configclient.AIProvider{
		ID:      "xiaomi-mimo-tts",
		Type:    "mimo-tts",
		BaseURL: srv.URL + "/v1",
		APIKey:  "sk-mimo",
		Model:   "mimo-v2.5-tts",
		Voice:   "冰糖",
	}, "人")
	require.NoError(t, err)
	require.Equal(t, []byte("ID3fake"), audio)
	require.Equal(t, "/v1/chat/completions", gotPath)
	require.Equal(t, "sk-mimo", gotAPIKey)
	require.Contains(t, gotBody, `"model":"mimo-v2.5-tts"`)
	require.Contains(t, gotBody, `"content":"人"`)
	require.Contains(t, gotBody, `"format":"mp3"`)
	require.Contains(t, gotBody, `"voice":"冰糖"`)
}
