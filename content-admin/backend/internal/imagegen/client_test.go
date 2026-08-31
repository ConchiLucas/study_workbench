package imagegen_test

import (
	"testing"

	"github.com/conchi/study-content-admin/internal/configclient"
	"github.com/conchi/study-content-admin/internal/imagegen"
	"github.com/stretchr/testify/require"
)

func TestPickProviderPrefersExecutable(t *testing.T) {
	ai := configclient.AIConfiguration{
		ActiveProviderID: "text",
		Providers: []configclient.AIProvider{
			{ID: "text", Type: "openai-compatible", BaseURL: "https://x/v1", Model: "t", Enabled: true, Capabilities: []string{"TEXT_GENERATION"}},
			{ID: "antigravity-gemini-image", Type: "openai-compatible", BaseURL: "https://img/v1", Model: "gemini-image", Enabled: true, Capabilities: []string{"IMAGE_GENERATION"}, APIKey: "k"},
		},
	}
	p, err := imagegen.PickProvider(ai)
	require.NoError(t, err)
	require.Equal(t, "antigravity-gemini-image", p.ID)
}
