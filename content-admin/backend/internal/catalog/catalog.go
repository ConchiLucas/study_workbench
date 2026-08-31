package catalog

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/conchi/study-content-admin/internal/configclient"
)

type CatalogProvider struct {
	ID           string            `json:"id"`
	Label        string            `json:"label"`
	Type         string            `json:"type"`
	BaseURL      string            `json:"baseUrl"`
	APIKey       string            `json:"apiKey"`
	Model        string            `json:"model"`
	MaxTokens    int               `json:"maxTokens"`
	Voice        string            `json:"voice,omitempty"`
	Capabilities []string          `json:"capabilities"`
	Options      map[string]string `json:"options"`
	Enabled      bool              `json:"enabled"`
	Active       bool              `json:"active"`
}

type CatalogAI struct {
	Active    string            `json:"active"`
	Providers []CatalogProvider `json:"providers"`
}

type CatalogDatabase struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Environment string            `json:"environment"`
	Host        string            `json:"host"`
	Port        int               `json:"port"`
	Database    string            `json:"database"`
	Username    string            `json:"username"`
	Password    string            `json:"password"`
	Parameters  map[string]string `json:"parameters,omitempty"`
	Active      bool              `json:"active"`
}

type CatalogCLIItem struct {
	ID               string   `json:"id"`
	Label            string   `json:"label"`
	Enabled          bool     `json:"enabled"`
	Command          string   `json:"command"`
	DefaultArgs      []string `json:"defaultArgs"`
	Model            string   `json:"model,omitempty"`
	ReasoningEffort  string   `json:"reasoningEffort,omitempty"`
	WorkingDirectory string   `json:"workingDirectory"`
	TimeoutSeconds   int      `json:"timeoutSeconds"`
	Capabilities     []string `json:"capabilities"`
	Active           bool     `json:"active"`
}

type CatalogCLI struct {
	Active  string           `json:"active"`
	Configs []CatalogCLIItem `json:"configs"`
}

type CatalogMinio struct {
	Configured      bool   `json:"configured"`
	Enabled         bool   `json:"enabled"`
	Endpoint        string `json:"endpoint"`
	AccessKeyID     string `json:"accessKeyId"`
	SecretAccessKey string `json:"secretAccessKey"`
	UseSSL          bool   `json:"useSsl"`
	BucketName      string `json:"bucketName"`
	BasePath        string `json:"basePath"`
}

type CatalogRuntime struct {
	SchemaVersion string `json:"schemaVersion"`
	GeneratedAt   string `json:"generatedAt"`
	JSON          string `json:"json"`
}

type Catalog struct {
	Databases     []CatalogDatabase `json:"databases"`
	AI            CatalogAI         `json:"ai"`
	LocalCLI      CatalogCLI        `json:"localCli"`
	ObjectStorage CatalogMinio      `json:"objectStorage"`
	ImageModels   CatalogAI         `json:"imageModels"`
	VideoModels   CatalogAI         `json:"videoModels"`
	VoiceModels   CatalogAI         `json:"voiceModels"`
	Runtime       CatalogRuntime    `json:"runtime"`
}

type Service struct {
	client *configclient.Client
}

func NewService(client *configclient.Client) *Service {
	return &Service{client: client}
}

func (s *Service) Build(ctx context.Context, refresh bool) (Catalog, error) {
	var (
		snapshot configclient.RuntimeConfiguration
		err      error
	)
	if refresh {
		snapshot, err = s.client.Refresh(ctx)
	} else {
		snapshot, err = s.client.Require(ctx)
	}
	if err != nil {
		return Catalog{}, err
	}

	ai := toCatalogAI(snapshot.AI)
	localCLI := toCatalogCLI(snapshot.LocalCLI)
	databases := toCatalogDatabases(snapshot.Databases)
	objectStorage := toCatalogMinio(snapshot.ObjectStorage)
	imageModels := s.loadModelSection(ctx, snapshot, "IMAGE_GENERATION")
	videoModels := s.loadModelSection(ctx, snapshot, "VIDEO_GENERATION")
	voiceModels := s.loadModelSection(ctx, snapshot, "AUDIO_TTS")
	runtime := toCatalogRuntime(snapshot)

	return Catalog{
		Databases:     databases,
		AI:            ai,
		LocalCLI:      localCLI,
		ObjectStorage: objectStorage,
		ImageModels:   imageModels,
		VideoModels:   videoModels,
		VoiceModels:   voiceModels,
		Runtime:       runtime,
	}, nil
}

func (s *Service) loadModelSection(ctx context.Context, snapshot configclient.RuntimeConfiguration, capability string) CatalogAI {
	var (
		section configclient.AIConfiguration
		err     error
	)
	switch capability {
	case "IMAGE_GENERATION":
		section, err = s.client.LoadImageModels(ctx)
	case "VIDEO_GENERATION":
		section, err = s.client.LoadVideoModels(ctx)
	case "AUDIO_TTS":
		section, err = s.client.LoadVoiceModels(ctx)
	default:
		return filterProvidersByCapability(snapshot.AI, capability)
	}
	if err != nil || len(section.Providers) == 0 {
		return filterProvidersByCapability(snapshot.AI, capability)
	}
	return toCatalogAI(section)
}

func filterProvidersByCapability(ai configclient.AIConfiguration, want string) CatalogAI {
	filtered := configclient.AIConfiguration{
		ActiveProviderID: ai.ActiveProviderID,
		Providers:        nil,
	}
	for _, p := range ai.Providers {
		if !p.Enabled {
			continue
		}
		for _, cap := range p.Capabilities {
			if cap == want {
				filtered.Providers = append(filtered.Providers, p)
				break
			}
		}
	}
	if len(filtered.Providers) == 0 {
		filtered.ActiveProviderID = ""
	} else {
		found := false
		for _, p := range filtered.Providers {
			if p.ID == filtered.ActiveProviderID {
				found = true
				break
			}
		}
		if !found {
			filtered.ActiveProviderID = preferredProviderID(want, filtered.Providers)
		}
	}
	return toCatalogAI(filtered)
}

func preferredProviderID(capability string, providers []configclient.AIProvider) string {
	if len(providers) == 0 {
		return ""
	}
	if capability == "AUDIO_TTS" {
		for _, p := range providers {
			typeName := strings.ToLower(strings.TrimSpace(p.Type))
			id := strings.ToLower(strings.TrimSpace(p.ID))
			if typeName == "grok-tts" || (strings.Contains(id, "grok") && strings.Contains(id, "tts")) {
				return p.ID
			}
		}
	}
	return providers[0].ID
}

func toCatalogAI(ai configclient.AIConfiguration) CatalogAI {
	out := CatalogAI{Active: ai.ActiveProviderID, Providers: make([]CatalogProvider, 0, len(ai.Providers))}
	for _, p := range ai.Providers {
		opts := p.Options
		if opts == nil {
			opts = map[string]string{}
		}
		caps := p.Capabilities
		if caps == nil {
			caps = []string{}
		}
		out.Providers = append(out.Providers, CatalogProvider{
			ID: p.ID, Label: p.Label, Type: p.Type, BaseURL: p.BaseURL, APIKey: p.APIKey,
			Model: p.Model, MaxTokens: p.MaxTokens, Voice: p.Voice, Capabilities: caps,
			Options: opts, Enabled: p.Enabled, Active: p.ID != "" && p.ID == ai.ActiveProviderID,
		})
	}
	return out
}

func toCatalogCLI(cli configclient.LocalCLIConfiguration) CatalogCLI {
	out := CatalogCLI{Active: cli.ActiveConfigID, Configs: make([]CatalogCLIItem, 0, len(cli.Configs))}
	for _, item := range cli.Configs {
		args := item.DefaultArgs
		if args == nil {
			args = []string{}
		}
		caps := item.Capabilities
		if caps == nil {
			caps = []string{}
		}
		out.Configs = append(out.Configs, CatalogCLIItem{
			ID: item.ID, Label: item.Label, Enabled: item.Enabled, Command: item.Command,
			DefaultArgs: args, Model: item.Model, ReasoningEffort: item.ReasoningEffort,
			WorkingDirectory: item.WorkingDirectory, TimeoutSeconds: item.TimeoutSeconds,
			Capabilities: caps, Active: item.ID != "" && item.ID == cli.ActiveConfigID,
		})
	}
	return out
}

func toCatalogDatabases(items []configclient.DatabaseConnection) []CatalogDatabase {
	out := make([]CatalogDatabase, 0, len(items))
	for _, item := range items {
		params := item.Parameters
		if params == nil {
			params = map[string]string{}
		}
		out = append(out, CatalogDatabase{
			ID: item.ID, Name: item.Name, Type: item.Type, Environment: item.Environment,
			Host: item.Host, Port: item.Port, Database: item.Database, Username: item.Username,
			Password: item.Password, Parameters: params, Active: false,
		})
	}
	return out
}

func toCatalogMinio(v configclient.ObjectStorageConfiguration) CatalogMinio {
	return CatalogMinio{
		Configured: v.Configured, Enabled: v.Enabled, Endpoint: v.Endpoint,
		AccessKeyID: v.AccessKeyID, SecretAccessKey: v.SecretAccessKey, UseSSL: v.UseSSL,
		BucketName: v.BucketName, BasePath: v.BasePath,
	}
}

func toCatalogRuntime(snapshot configclient.RuntimeConfiguration) CatalogRuntime {
	raw, _ := json.MarshalIndent(snapshot, "", "  ")
	generated := snapshot.GeneratedAt.UTC().Format(time.RFC3339)
	if snapshot.GeneratedAt.IsZero() {
		generated = ""
	}
	return CatalogRuntime{
		SchemaVersion: snapshot.SchemaVersion,
		GeneratedAt:   generated,
		JSON:          string(raw),
	}
}
