package english

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"gorm.io/gorm"

	"github.com/conchi/study-content-admin/internal/configclient"
	"github.com/conchi/study-content-admin/internal/sense"
	"github.com/conchi/study-content-admin/internal/storage"
	"github.com/conchi/study-content-admin/internal/tts"
)

type Asset struct {
	KpID                    int64     `gorm:"column:kp_id;primaryKey" json:"kpId"`
	WordText                string    `gorm:"column:word_text" json:"wordText"`
	ModuleCode              string    `gorm:"column:module_code" json:"moduleCode"`
	ModuleName              string    `gorm:"column:module_name" json:"moduleName"`
	ModuleOrder             int       `gorm:"column:module_order" json:"moduleOrder"`
	KpOrder                 int       `gorm:"column:kp_order" json:"kpOrder"`
	NeedsSenseImage         bool      `gorm:"column:needs_sense_image" json:"needsSenseImage"`
	NeedsSenseImageOverride *bool     `gorm:"column:needs_sense_image_override" json:"needsSenseImageOverride"`
	GlyphImageURL           string    `gorm:"column:glyph_image_url" json:"glyphImageUrl"`
	SenseImageURL           string    `gorm:"column:sense_image_url" json:"senseImageUrl"`
	SpeechAudioURL          string    `gorm:"column:speech_audio_url" json:"speechAudioUrl"`
	SyncedAt                time.Time `gorm:"column:synced_at" json:"syncedAt"`
	UpdatedAt               time.Time `gorm:"column:updated_at" json:"updatedAt"`
}

func (Asset) TableName() string { return "english_assets" }

func (a Asset) EffectiveNeedsSenseImage() bool {
	if a.NeedsSenseImageOverride != nil {
		return *a.NeedsSenseImageOverride
	}
	return a.NeedsSenseImage
}

type WordDTO struct {
	KpID                     int64  `json:"kpId"`
	WordText                 string `json:"wordText"`
	ModuleCode               string `json:"moduleCode"`
	ModuleName               string `json:"moduleName"`
	ModuleOrder              int    `json:"moduleOrder"`
	KpOrder                  int    `json:"kpOrder"`
	NeedsSenseImage          bool   `json:"needsSenseImage"`
	NeedsSenseImageOverride  *bool  `json:"needsSenseImageOverride"`
	EffectiveNeedsSenseImage bool   `json:"effectiveNeedsSenseImage"`
	GlyphImageURL            string `json:"glyphImageUrl"`
	SenseImageURL            string `json:"senseImageUrl"`
	SpeechAudioURL           string `json:"speechAudioUrl"`
}

type GroupDTO struct {
	ModuleCode  string    `json:"moduleCode"`
	ModuleName  string    `json:"moduleName"`
	ModuleOrder int       `json:"moduleOrder"`
	Words       []WordDTO `json:"words"`
}

type ListResult struct {
	View   string     `json:"view"`
	Total  int        `json:"total"`
	Groups []GroupDTO `json:"groups,omitempty"`
	Words  []WordDTO  `json:"words,omitempty"`
}

type SyncResult struct {
	Upserted int `json:"upserted"`
	Total    int `json:"total"`
}

type Service struct {
	db       *gorm.DB
	store    ObjectPutter
	renderer GlyphRenderer
	images   ImageGenerator
	cfg      ImageModelSource
	voices   VoiceModelSource
	speech   SpeechSynthesizer
}

type ObjectPutter interface {
	PutPNG(ctx context.Context, objectKey string, png []byte) (string, error)
	GetPNG(ctx context.Context, relativeOrFullKey string) ([]byte, error)
	PutBytes(ctx context.Context, objectKey string, data []byte, contentType string) (string, error)
	GetBytes(ctx context.Context, relativeOrFullKey string) ([]byte, error)
	EnglishGlyphKey(kpID int64) string
	EnglishSenseKey(kpID int64) string
	EnglishSpeechKey(kpID int64) string
}

type GlyphRenderer interface {
	RenderEnglishPNG(word string) ([]byte, error)
}

type ImageGenerator interface {
	GeneratePNG(ctx context.Context, provider configclient.AIProvider, prompt, negative string) ([]byte, error)
}

type ImageModelSource interface {
	LoadImageModels(ctx context.Context) (configclient.AIConfiguration, error)
}

type VoiceModelSource interface {
	LoadVoiceModels(ctx context.Context) (configclient.AIConfiguration, error)
}

type SpeechSynthesizer interface {
	Synthesize(ctx context.Context, provider configclient.AIProvider, text string) ([]byte, error)
}

func NewService(db *gorm.DB, store ObjectPutter, renderer GlyphRenderer, images ImageGenerator, cfg ImageModelSource, voices VoiceModelSource, speech SpeechSynthesizer) *Service {
	return &Service{db: db, store: store, renderer: renderer, images: images, cfg: cfg, voices: voices, speech: speech}
}

func publicBase() string {
	base := strings.TrimRight(os.Getenv("APP_PUBLIC_BASE"), "/")
	if base == "" {
		base = "http://localhost:19091"
	}
	return base
}

func pickSpeechProvider(ai configclient.AIConfiguration, asset Asset) (configclient.AIProvider, error) {
	_ = asset
	provider, err := tts.PickProvider(ai)
	if err != nil {
		return provider, err
	}
	return tts.WithLanguage(provider, "en"), nil
}

func speechPublicURL(kpID int64) string {
	return fmt.Sprintf("%s/api/v1/english/words/%d/speech.mp3", publicBase(), kpID)
}

func (s *Service) synthesizeSpeech(ctx context.Context, asset Asset) ([]byte, error) {
	if s.voices == nil || s.speech == nil {
		return nil, fmt.Errorf("TTS 未就绪（检查共享配置中心）")
	}
	text := strings.TrimSpace(asset.WordText)
	if text == "" {
		return nil, fmt.Errorf("单词为空")
	}
	ai, err := s.voices.LoadVoiceModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("加载语音配置失败: %w", err)
	}
	provider, err := pickSpeechProvider(ai, asset)
	if err != nil {
		return nil, err
	}
	return s.speech.Synthesize(ctx, provider, text)
}

func (s *Service) storeSpeech(ctx context.Context, asset *Asset, mp3 []byte) error {
	if s.store == nil {
		return fmt.Errorf("语音存储未就绪（检查 MinIO）")
	}
	if _, err := s.store.PutBytes(ctx, storage.EnglishSpeechObjectKey(asset.KpID), mp3, "audio/mpeg"); err != nil {
		return err
	}
	asset.SpeechAudioURL = speechPublicURL(asset.KpID)
	asset.UpdatedAt = time.Now().UTC()
	return s.db.Save(asset).Error
}

// SpeechMP3 returns cached MinIO audio when present; otherwise synthesizes, stores, and returns bytes.
func (s *Service) SpeechMP3(ctx context.Context, kpID int64) ([]byte, error) {
	var asset Asset
	if err := s.db.First(&asset, "kp_id = ?", kpID).Error; err != nil {
		return nil, err
	}
	if s.store != nil && strings.TrimSpace(asset.SpeechAudioURL) != "" {
		if data, err := s.store.GetBytes(ctx, s.store.EnglishSpeechKey(kpID)); err == nil && len(data) > 0 {
			return data, nil
		}
	}
	mp3, err := s.synthesizeSpeech(ctx, asset)
	if err != nil {
		return nil, err
	}
	if err := s.storeSpeech(ctx, &asset, mp3); err != nil {
		return nil, err
	}
	return mp3, nil
}

// RegenerateSpeech always synthesizes and overwrites MinIO + DB URL.
func (s *Service) RegenerateSpeech(ctx context.Context, kpID int64) (WordDTO, error) {
	var asset Asset
	if err := s.db.First(&asset, "kp_id = ?", kpID).Error; err != nil {
		return WordDTO{}, err
	}
	mp3, err := s.synthesizeSpeech(ctx, asset)
	if err != nil {
		return WordDTO{}, err
	}
	if err := s.storeSpeech(ctx, &asset, mp3); err != nil {
		return WordDTO{}, err
	}
	return toDTO(asset), nil
}

// BatchGenerateSpeech generates missing speech for one module only.
func (s *Service) BatchGenerateSpeech(ctx context.Context, moduleCode string) (GlyphBatchResult, error) {
	moduleCode = strings.TrimSpace(moduleCode)
	if moduleCode == "" {
		return GlyphBatchResult{}, fmt.Errorf("moduleCode 不能为空")
	}
	if s.store == nil || s.voices == nil || s.speech == nil {
		return GlyphBatchResult{}, fmt.Errorf("TTS 未就绪（检查 MinIO 与语音配置）")
	}
	var assets []Asset
	if err := s.db.Where("module_code = ? AND (speech_audio_url = '' OR speech_audio_url IS NULL)", moduleCode).
		Order("module_order ASC, kp_order ASC, kp_id ASC").
		Find(&assets).Error; err != nil {
		return GlyphBatchResult{}, err
	}
	out := GlyphBatchResult{}
	for i, asset := range assets {
		if strings.TrimSpace(asset.SpeechAudioURL) != "" {
			out.Skipped++
			continue
		}
		mp3, err := s.synthesizeSpeech(ctx, asset)
		if err != nil {
			out.Failed++
			if len(out.Errors) < 10 {
				out.Errors = append(out.Errors, fmt.Sprintf("%s(%d): %v", asset.WordText, asset.KpID, err))
			}
			continue
		}
		if err := s.storeSpeech(ctx, &asset, mp3); err != nil {
			out.Failed++
			if len(out.Errors) < 10 {
				out.Errors = append(out.Errors, fmt.Sprintf("%s(%d): %v", asset.WordText, asset.KpID, err))
			}
			continue
		}
		out.Generated++
		if i+1 < len(assets) {
			select {
			case <-ctx.Done():
				return out, ctx.Err()
			case <-time.After(500 * time.Millisecond):
			}
		}
	}
	return out, nil
}

type kpRow struct {
	ID          int64
	Title       string
	KpOrder     int
	ModuleCode  string
	ModuleName  string
	ModuleOrder int
}

func (s *Service) Sync() (SyncResult, error) {
	var rows []kpRow
	err := s.db.Raw(`
		SELECT kp.id AS id, kp.title AS title, kp.order_no AS kp_order,
		       m.code AS module_code, m.name AS module_name, m.order_no AS module_order
		FROM knowledge_points kp
		JOIN modules m ON m.id = kp.module_id
		JOIN subjects s ON s.id = m.subject_id
		WHERE s.code = ?
		ORDER BY m.order_no, kp.order_no, kp.id
	`, "english").Scan(&rows).Error
	if err != nil {
		return SyncResult{}, err
	}

	now := time.Now().UTC()
	upserted := 0
	for _, row := range rows {
		char := row.Title
		system := NeedsSenseImage(char)
		var existing Asset
		findErr := s.db.First(&existing, "kp_id = ?", row.ID).Error
		if findErr == gorm.ErrRecordNotFound {
			asset := Asset{
				KpID:            row.ID,
				WordText:        char,
				ModuleCode:      row.ModuleCode,
				ModuleName:      row.ModuleName,
				ModuleOrder:     row.ModuleOrder,
				KpOrder:         row.KpOrder,
				NeedsSenseImage: system,
				SyncedAt:        now,
				UpdatedAt:       now,
			}
			if err := s.db.Create(&asset).Error; err != nil {
				return SyncResult{}, err
			}
			upserted++
			continue
		}
		if findErr != nil {
			return SyncResult{}, findErr
		}
		existing.WordText = char
		existing.ModuleCode = row.ModuleCode
		existing.ModuleName = row.ModuleName
		existing.ModuleOrder = row.ModuleOrder
		existing.KpOrder = row.KpOrder
		existing.NeedsSenseImage = system
		existing.SyncedAt = now
		existing.UpdatedAt = now
		if err := s.db.Save(&existing).Error; err != nil {
			return SyncResult{}, err
		}
		upserted++
	}
	return SyncResult{Upserted: upserted, Total: len(rows)}, nil
}

func (s *Service) List(view string, needsFilter *bool) (ListResult, error) {
	if view == "" {
		view = "groups"
	}
	var assets []Asset
	if err := s.db.Order("module_order ASC, kp_order ASC, kp_id ASC").Find(&assets).Error; err != nil {
		return ListResult{}, err
	}

	dtos := make([]WordDTO, 0, len(assets))
	for _, a := range assets {
		dto := toDTO(a)
		if needsFilter != nil && dto.EffectiveNeedsSenseImage != *needsFilter {
			continue
		}
		dtos = append(dtos, dto)
	}

	out := ListResult{View: view, Total: len(dtos)}
	if view == "table" {
		out.Words = dtos
		return out, nil
	}

	byMod := map[string]*GroupDTO{}
	order := []string{}
	for _, dto := range dtos {
		g, ok := byMod[dto.ModuleCode]
		if !ok {
			g = &GroupDTO{
				ModuleCode:  dto.ModuleCode,
				ModuleName:  dto.ModuleName,
				ModuleOrder: dto.ModuleOrder,
				Words:       []WordDTO{},
			}
			byMod[dto.ModuleCode] = g
			order = append(order, dto.ModuleCode)
		}
		g.Words = append(g.Words, dto)
	}
	out.Groups = make([]GroupDTO, 0, len(order))
	for _, code := range order {
		out.Groups = append(out.Groups, *byMod[code])
	}
	return out, nil
}

func (s *Service) PatchOverride(kpID int64, override *bool) (WordDTO, error) {
	var asset Asset
	if err := s.db.First(&asset, "kp_id = ?", kpID).Error; err != nil {
		return WordDTO{}, err
	}
	asset.NeedsSenseImageOverride = override
	asset.UpdatedAt = time.Now().UTC()
	if err := s.db.Save(&asset).Error; err != nil {
		return WordDTO{}, err
	}
	return toDTO(asset), nil
}

type GlyphBatchResult struct {
	Generated int      `json:"generated"`
	Skipped   int      `json:"skipped"`
	Failed    int      `json:"failed"`
	Retried   int      `json:"retried,omitempty"`
	Workers   int      `json:"workers,omitempty"`
	Errors    []string `json:"errors,omitempty"`
}

func (s *Service) GenerateGlyph(ctx context.Context, kpID int64) (WordDTO, error) {
	if s.store == nil || s.renderer == nil {
		return WordDTO{}, fmt.Errorf("字图生成未就绪（检查 MinIO 与字体）")
	}
	var asset Asset
	if err := s.db.First(&asset, "kp_id = ?", kpID).Error; err != nil {
		return WordDTO{}, err
	}
	png, err := s.renderer.RenderEnglishPNG(asset.WordText)
	if err != nil {
		return WordDTO{}, err
	}
	key, err := s.store.PutPNG(ctx, storage.EnglishGlyphObjectKey(kpID), png)
	if err != nil {
		return WordDTO{}, err
	}
	// Browser-facing URL is proxied by this service (MinIO bucket is private).
	publicBase := strings.TrimRight(os.Getenv("APP_PUBLIC_BASE"), "/")
	if publicBase == "" {
		publicBase = "http://localhost:19091"
	}
	asset.GlyphImageURL = fmt.Sprintf("%s/api/v1/english/words/%d/glyph.png", publicBase, kpID)
	_ = key
	asset.UpdatedAt = time.Now().UTC()
	if err := s.db.Save(&asset).Error; err != nil {
		return WordDTO{}, err
	}
	return toDTO(asset), nil
}

func (s *Service) GlyphPNG(ctx context.Context, kpID int64) ([]byte, error) {
	if s.store == nil {
		return nil, fmt.Errorf("字图存储未就绪")
	}
	var asset Asset
	if err := s.db.First(&asset, "kp_id = ?", kpID).Error; err != nil {
		return nil, err
	}
	if asset.GlyphImageURL == "" {
		return nil, fmt.Errorf("尚未生成字图")
	}
	return s.store.GetPNG(ctx, s.store.EnglishGlyphKey(kpID))
}

func (s *Service) BatchGenerateGlyphs(ctx context.Context, force bool) (GlyphBatchResult, error) {
	if s.store == nil || s.renderer == nil {
		return GlyphBatchResult{}, fmt.Errorf("字图生成未就绪（检查 MinIO 与字体）")
	}
	q := s.db.Order("module_order ASC, kp_order ASC, kp_id ASC")
	if !force {
		q = q.Where("glyph_image_url = '' OR glyph_image_url IS NULL")
	}
	var assets []Asset
	if err := q.Find(&assets).Error; err != nil {
		return GlyphBatchResult{}, err
	}
	out := GlyphBatchResult{}
	for _, asset := range assets {
		if _, err := s.GenerateGlyph(ctx, asset.KpID); err != nil {
			out.Failed++
			if len(out.Errors) < 10 {
				out.Errors = append(out.Errors, fmt.Sprintf("%s(%d): %v", asset.WordText, asset.KpID, err))
			}
			continue
		}
		out.Generated++
	}
	return out, nil
}

func (s *Service) GenerateSense(ctx context.Context, kpID int64) (WordDTO, error) {
	if s.store == nil || s.images == nil || s.cfg == nil {
		return WordDTO{}, fmt.Errorf("义图生成未就绪（检查 MinIO 与图片模型配置）")
	}
	var asset Asset
	if err := s.db.First(&asset, "kp_id = ?", kpID).Error; err != nil {
		return WordDTO{}, err
	}
	if !asset.EffectiveNeedsSenseImage() {
		return WordDTO{}, fmt.Errorf("该词按规则不需要义图")
	}
	ai, err := s.cfg.LoadImageModels(ctx)
	if err != nil {
		return WordDTO{}, fmt.Errorf("加载图片模型失败: %w", err)
	}
	providers := listImageProviders(ai)
	if len(providers) == 0 {
		return WordDTO{}, fmt.Errorf("配置中心没有可用的图片模型")
	}
	prompt := sense.PromptEnglish(asset.WordText)
	var png []byte
	var lastErr error
	for _, provider := range providers {
		png, lastErr = s.images.GeneratePNG(ctx, provider, prompt, sense.Negative)
		if lastErr == nil {
			break
		}
	}
	if lastErr != nil {
		return WordDTO{}, lastErr
	}
	if _, err := s.store.PutPNG(ctx, storage.EnglishSenseObjectKey(kpID), png); err != nil {
		return WordDTO{}, err
	}
	publicBase := strings.TrimRight(os.Getenv("APP_PUBLIC_BASE"), "/")
	if publicBase == "" {
		publicBase = "http://localhost:19091"
	}
	asset.SenseImageURL = fmt.Sprintf("%s/api/v1/english/words/%d/sense.png", publicBase, kpID)
	asset.UpdatedAt = time.Now().UTC()
	if err := s.db.Save(&asset).Error; err != nil {
		return WordDTO{}, err
	}
	return toDTO(asset), nil
}

func (s *Service) SensePNG(ctx context.Context, kpID int64) ([]byte, error) {
	if s.store == nil {
		return nil, fmt.Errorf("义图存储未就绪")
	}
	var asset Asset
	if err := s.db.First(&asset, "kp_id = ?", kpID).Error; err != nil {
		return nil, err
	}
	if asset.SenseImageURL == "" {
		return nil, fmt.Errorf("尚未生成义图")
	}
	return s.store.GetPNG(ctx, s.store.EnglishSenseKey(kpID))
}

func (s *Service) BatchGenerateSenses(ctx context.Context, moduleCode string, workers, maxRetries int) (GlyphBatchResult, error) {
	if s.store == nil || s.images == nil || s.cfg == nil {
		return GlyphBatchResult{}, fmt.Errorf("义图生成未就绪（检查 MinIO 与图片模型配置）")
	}
	workers = clampInt(workers, 1, 8, senseWorkerDefault())
	// maxRetries: how many times a failed job may re-enter the retry queue (hard cap 3).
	maxRetries = clampInt(maxRetries, 0, 3, 3)

	q := s.db.Where("(sense_image_url = '' OR sense_image_url IS NULL)")
	if moduleCode != "" {
		q = q.Where("module_code = ?", moduleCode)
	}
	var assets []Asset
	if err := q.Order("module_order ASC, kp_order ASC, kp_id ASC").Find(&assets).Error; err != nil {
		return GlyphBatchResult{}, err
	}

	out := GlyphBatchResult{Workers: workers}
	type job struct {
		kpID int64
		char string
	}
	pending := make([]job, 0, len(assets))
	for _, asset := range assets {
		if !asset.EffectiveNeedsSenseImage() {
			out.Skipped++
			continue
		}
		pending = append(pending, job{kpID: asset.KpID, char: asset.WordText})
	}

	var retried int
	for attempt := 0; len(pending) > 0; attempt++ {
		if attempt > maxRetries {
			break
		}
		if attempt > 0 {
			retried += len(pending)
			// Backoff before retrying the failure queue (rate limits / transient 502).
			backoff := time.Duration(attempt*attempt) * 5 * time.Second
			select {
			case <-ctx.Done():
				return out, ctx.Err()
			case <-time.After(backoff):
			}
		}

		type failItem struct {
			job job
			err error
		}
		jobs := make(chan job, len(pending))
		fails := make(chan failItem, len(pending))
		var generated atomic.Int64

		var wg sync.WaitGroup
		for w := 0; w < workers; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := range jobs {
					if ctx.Err() != nil {
						fails <- failItem{job: j, err: ctx.Err()}
						continue
					}
					if _, err := s.GenerateSense(ctx, j.kpID); err != nil {
						fails <- failItem{job: j, err: err}
						continue
					}
					generated.Add(1)
				}
			}()
		}
		for _, j := range pending {
			jobs <- j
		}
		close(jobs)
		wg.Wait()
		close(fails)

		out.Generated += int(generated.Load())
		next := make([]job, 0)
		for f := range fails {
			next = append(next, f.job)
			if attempt == maxRetries && len(out.Errors) < 20 {
				out.Errors = append(out.Errors, fmt.Sprintf("%s(%d): %v", f.job.char, f.job.kpID, f.err))
			}
		}
		pending = next
		if len(pending) == 0 {
			break
		}
	}
	out.Failed = len(pending)
	out.Retried = retried
	return out, nil
}

func senseWorkerDefault() int {
	if v := strings.TrimSpace(os.Getenv("SENSE_WORKERS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return 3
}

func clampInt(v, min, max, def int) int {
	if v == 0 {
		v = def
	}
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func listImageProviders(ai configclient.AIConfiguration) []configclient.AIProvider {
	var first, active, gemini, grok []configclient.AIProvider
	seen := map[string]bool{}
	add := func(dst *[]configclient.AIProvider, p configclient.AIProvider) {
		if seen[p.ID] {
			return
		}
		seen[p.ID] = true
		*dst = append(*dst, p)
	}
	for i := range ai.Providers {
		p := ai.Providers[i]
		if !p.Enabled || !strings.EqualFold(p.Type, "openai-compatible") {
			continue
		}
		if strings.TrimSpace(p.BaseURL) == "" || strings.TrimSpace(p.Model) == "" {
			continue
		}
		hasGen := false
		for _, c := range p.Capabilities {
			if c == "IMAGE_GENERATION" {
				hasGen = true
				break
			}
		}
		if !hasGen {
			continue
		}
		idLower := strings.ToLower(p.ID)
		modelLower := strings.ToLower(p.Model)
		switch {
		case strings.Contains(idLower, "gemini"):
			add(&gemini, p)
		case strings.Contains(idLower, "grok") || strings.HasPrefix(modelLower, "grok-imagine-image"):
			add(&grok, p)
		case p.ID == ai.ActiveProviderID:
			add(&active, p)
		default:
			add(&first, p)
		}
	}
	// Prefer Grok when Gemini quota is often the bottleneck for large batches.
	out := append([]configclient.AIProvider{}, grok...)
	out = append(out, gemini...)
	out = append(out, active...)
	out = append(out, first...)
	return out
}

func pickImageProvider(ai configclient.AIConfiguration) (configclient.AIProvider, error) {
	providers := listImageProviders(ai)
	if len(providers) == 0 {
		return configclient.AIProvider{}, fmt.Errorf("配置中心没有可用的图片模型")
	}
	return providers[0], nil
}

func toDTO(a Asset) WordDTO {
	glyph := withCacheBust(a.GlyphImageURL, a.UpdatedAt)
	senseURL := withCacheBust(a.SenseImageURL, a.UpdatedAt)
	speechURL := withCacheBust(a.SpeechAudioURL, a.UpdatedAt)
	return WordDTO{
		KpID:                     a.KpID,
		WordText:                 a.WordText,
		ModuleCode:               a.ModuleCode,
		ModuleName:               a.ModuleName,
		ModuleOrder:              a.ModuleOrder,
		KpOrder:                  a.KpOrder,
		NeedsSenseImage:          a.NeedsSenseImage,
		NeedsSenseImageOverride:  a.NeedsSenseImageOverride,
		EffectiveNeedsSenseImage: a.EffectiveNeedsSenseImage(),
		GlyphImageURL:            glyph,
		SenseImageURL:            senseURL,
		SpeechAudioURL:           speechURL,
	}
}

func withCacheBust(url string, updatedAt time.Time) string {
	if url == "" || updatedAt.IsZero() {
		return url
	}
	sep := "?"
	if strings.Contains(url, "?") {
		sep = "&"
	}
	return fmt.Sprintf("%s%sv=%d", url, sep, updatedAt.Unix())
}
