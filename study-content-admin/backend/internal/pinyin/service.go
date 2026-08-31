package pinyin

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/conchi/study-content-admin/internal/configclient"
	"github.com/conchi/study-content-admin/internal/storage"
	"github.com/conchi/study-content-admin/internal/tts"
)

type Asset struct {
	KpID          int64     `gorm:"column:kp_id;primaryKey" json:"kpId"`
	Letter        string    `gorm:"column:letter" json:"letter"`
	ModuleCode    string    `gorm:"column:module_code" json:"moduleCode"`
	ModuleName    string    `gorm:"column:module_name" json:"moduleName"`
	ModuleOrder   int       `gorm:"column:module_order" json:"moduleOrder"`
	KpOrder       int       `gorm:"column:kp_order" json:"kpOrder"`
	SoloText      string    `gorm:"column:solo_text" json:"soloText"`
	WordText      string    `gorm:"column:word_text" json:"wordText"`
	SoloSpeechURL string    `gorm:"column:solo_speech_url" json:"soloSpeechUrl"`
	WordSpeechURL string    `gorm:"column:word_speech_url" json:"wordSpeechUrl"`
	GlyphImageURL string    `gorm:"column:glyph_image_url" json:"glyphImageUrl"`
	SyncedAt      time.Time `gorm:"column:synced_at" json:"syncedAt"`
	UpdatedAt     time.Time `gorm:"column:updated_at" json:"updatedAt"`
}

func (Asset) TableName() string { return "pinyin_assets" }

type ItemDTO struct {
	KpID          int64  `json:"kpId"`
	Letter        string `json:"letter"`
	ModuleCode    string `json:"moduleCode"`
	ModuleName    string `json:"moduleName"`
	ModuleOrder   int    `json:"moduleOrder"`
	KpOrder       int    `json:"kpOrder"`
	SoloText      string `json:"soloText"`
	WordText      string `json:"wordText"`
	SoloSpeechURL string `json:"soloSpeechUrl"`
	WordSpeechURL string `json:"wordSpeechUrl"`
	GlyphImageURL string `json:"glyphImageUrl"`
}

type GroupDTO struct {
	ModuleCode  string    `json:"moduleCode"`
	ModuleName  string    `json:"moduleName"`
	ModuleOrder int       `json:"moduleOrder"`
	Items       []ItemDTO `json:"items"`
}

type ListResult struct {
	View   string     `json:"view"`
	Total  int        `json:"total"`
	Groups []GroupDTO `json:"groups,omitempty"`
	Items  []ItemDTO  `json:"items,omitempty"`
}

type SyncResult struct {
	Upserted int `json:"upserted"`
	Total    int `json:"total"`
}

type BatchResult struct {
	Generated int      `json:"generated"`
	Skipped   int      `json:"skipped"`
	Failed    int      `json:"failed"`
	Errors    []string `json:"errors,omitempty"`
}

type ObjectPutter interface {
	PutBytes(ctx context.Context, objectKey string, data []byte, contentType string) (string, error)
	GetBytes(ctx context.Context, relativeOrFullKey string) ([]byte, error)
}

type GlyphRenderer interface {
	RenderPNG(text string) ([]byte, error)
}

type VoiceModelSource interface {
	LoadVoiceModels(ctx context.Context) (configclient.AIConfiguration, error)
}

type SpeechSynthesizer interface {
	Synthesize(ctx context.Context, provider configclient.AIProvider, text string) ([]byte, error)
}

type Service struct {
	db       *gorm.DB
	store    ObjectPutter
	voices   VoiceModelSource
	speech   SpeechSynthesizer
	renderer GlyphRenderer
}

func NewService(db *gorm.DB, store ObjectPutter, voices VoiceModelSource, speech SpeechSynthesizer, renderer GlyphRenderer) *Service {
	return &Service{db: db, store: store, voices: voices, speech: speech, renderer: renderer}
}

func publicBase() string {
	base := strings.TrimRight(os.Getenv("APP_PUBLIC_BASE"), "/")
	if base == "" {
		base = "http://localhost:19091"
	}
	return base
}

func soloPublicURL(kpID int64) string {
	return fmt.Sprintf("%s/api/v1/pinyin/items/%d/speech/solo.mp3", publicBase(), kpID)
}

func wordPublicURL(kpID int64) string {
	return fmt.Sprintf("%s/api/v1/pinyin/items/%d/speech/word.mp3", publicBase(), kpID)
}

func objectKeyFor(kind string, kpID int64) (string, error) {
	switch kind {
	case "solo":
		return storage.PinyinSoloObjectKey(kpID), nil
	case "word":
		return storage.PinyinWordObjectKey(kpID), nil
	default:
		return "", fmt.Errorf("kind 必须是 solo 或 word")
	}
}

func (s *Service) textFor(asset Asset, kind string) (string, error) {
	switch kind {
	case "solo":
		return strings.TrimSpace(asset.SoloText), nil
	case "word":
		return strings.TrimSpace(asset.WordText), nil
	default:
		return "", fmt.Errorf("kind 必须是 solo 或 word")
	}
}

func (s *Service) speechURL(asset *Asset, kind string) *string {
	switch kind {
	case "solo":
		return &asset.SoloSpeechURL
	case "word":
		return &asset.WordSpeechURL
	default:
		return nil
	}
}

func (s *Service) publicURL(kpID int64, kind string) string {
	if kind == "solo" {
		return soloPublicURL(kpID)
	}
	return wordPublicURL(kpID)
}

func (s *Service) synthesize(ctx context.Context, text string) ([]byte, error) {
	if s.voices == nil || s.speech == nil {
		return nil, fmt.Errorf("TTS 未就绪（检查共享配置中心）")
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("朗读文本为空")
	}
	ai, err := s.voices.LoadVoiceModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("加载语音配置失败: %w", err)
	}
	provider, err := tts.PickProvider(ai)
	if err != nil {
		return nil, err
	}
	return s.speech.Synthesize(ctx, provider, text)
}

func (s *Service) storeSpeech(ctx context.Context, asset *Asset, kind string, mp3 []byte) error {
	if s.store == nil {
		return fmt.Errorf("语音存储未就绪（检查 MinIO）")
	}
	key, err := objectKeyFor(kind, asset.KpID)
	if err != nil {
		return err
	}
	if _, err := s.store.PutBytes(ctx, key, mp3, "audio/mpeg"); err != nil {
		return err
	}
	urlPtr := s.speechURL(asset, kind)
	if urlPtr == nil {
		return fmt.Errorf("kind 必须是 solo 或 word")
	}
	*urlPtr = s.publicURL(asset.KpID, kind)
	asset.UpdatedAt = time.Now().UTC()
	return s.db.Save(asset).Error
}

// SpeechMP3 returns cached audio when present; otherwise synthesizes, stores, and returns bytes.
func (s *Service) SpeechMP3(ctx context.Context, kpID int64, kind string) ([]byte, error) {
	kind = strings.TrimSpace(kind)
	key, err := objectKeyFor(kind, kpID)
	if err != nil {
		return nil, err
	}
	var asset Asset
	if err := s.db.First(&asset, "kp_id = ?", kpID).Error; err != nil {
		return nil, err
	}
	urlPtr := s.speechURL(&asset, kind)
	if s.store != nil && urlPtr != nil && strings.TrimSpace(*urlPtr) != "" {
		if data, err := s.store.GetBytes(ctx, key); err == nil && len(data) > 0 {
			return data, nil
		}
	}
	text, err := s.textFor(asset, kind)
	if err != nil {
		return nil, err
	}
	if text == "" {
		return nil, fmt.Errorf("%s 文本为空，无法生成读音", kind)
	}
	mp3, err := s.synthesize(ctx, text)
	if err != nil {
		return nil, err
	}
	if err := s.storeSpeech(ctx, &asset, kind, mp3); err != nil {
		return nil, err
	}
	return mp3, nil
}

// RegenerateSpeech always synthesizes and overwrites MinIO + DB URL.
func (s *Service) RegenerateSpeech(ctx context.Context, kpID int64, kind string) (ItemDTO, error) {
	kind = strings.TrimSpace(kind)
	if _, err := objectKeyFor(kind, kpID); err != nil {
		return ItemDTO{}, err
	}
	var asset Asset
	if err := s.db.First(&asset, "kp_id = ?", kpID).Error; err != nil {
		return ItemDTO{}, err
	}
	text, err := s.textFor(asset, kind)
	if err != nil {
		return ItemDTO{}, err
	}
	if text == "" {
		return ItemDTO{}, fmt.Errorf("%s 文本为空，无法生成读音", kind)
	}
	mp3, err := s.synthesize(ctx, text)
	if err != nil {
		return ItemDTO{}, err
	}
	if err := s.storeSpeech(ctx, &asset, kind, mp3); err != nil {
		return ItemDTO{}, err
	}
	return toDTO(asset), nil
}

// BatchGenerateSpeech generates missing solo/word speech for one module.
func (s *Service) BatchGenerateSpeech(ctx context.Context, moduleCode string) (BatchResult, error) {
	moduleCode = strings.TrimSpace(moduleCode)
	if moduleCode == "" {
		return BatchResult{}, fmt.Errorf("moduleCode 不能为空")
	}
	if s.store == nil || s.voices == nil || s.speech == nil {
		return BatchResult{}, fmt.Errorf("TTS 未就绪（检查 MinIO 与语音配置）")
	}
	var assets []Asset
	if err := s.db.Where("module_code = ?", moduleCode).
		Order("module_order ASC, kp_order ASC, kp_id ASC").
		Find(&assets).Error; err != nil {
		return BatchResult{}, err
	}
	out := BatchResult{}
	needDelay := false
	for i := range assets {
		asset := &assets[i]
		for _, kind := range []string{"solo", "word"} {
			text, _ := s.textFor(*asset, kind)
			urlPtr := s.speechURL(asset, kind)
			if text == "" {
				out.Skipped++
				continue
			}
			if urlPtr != nil && strings.TrimSpace(*urlPtr) != "" {
				out.Skipped++
				continue
			}
			if needDelay {
				select {
				case <-ctx.Done():
					return out, ctx.Err()
				case <-time.After(400 * time.Millisecond):
				}
			}
			mp3, err := s.synthesize(ctx, text)
			if err != nil {
				out.Failed++
				if len(out.Errors) < 10 {
					out.Errors = append(out.Errors, fmt.Sprintf("%s/%s(%d): %v", asset.Letter, kind, asset.KpID, err))
				}
				needDelay = true
				continue
			}
			if err := s.storeSpeech(ctx, asset, kind, mp3); err != nil {
				out.Failed++
				if len(out.Errors) < 10 {
					out.Errors = append(out.Errors, fmt.Sprintf("%s/%s(%d): %v", asset.Letter, kind, asset.KpID, err))
				}
				needDelay = true
				continue
			}
			out.Generated++
			needDelay = true
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
	`, "pinyin").Scan(&rows).Error
	if err != nil {
		return SyncResult{}, err
	}

	now := time.Now().UTC()
	upserted := 0
	for _, row := range rows {
		letter := row.Title
		reading := Readings[letter]
		var existing Asset
		findErr := s.db.First(&existing, "kp_id = ?", row.ID).Error
		if findErr == gorm.ErrRecordNotFound {
			asset := Asset{
				KpID:        row.ID,
				Letter:      letter,
				ModuleCode:  row.ModuleCode,
				ModuleName:  row.ModuleName,
				ModuleOrder: row.ModuleOrder,
				KpOrder:     row.KpOrder,
				SoloText:    reading.Solo,
				WordText:    reading.Word,
				SyncedAt:    now,
				UpdatedAt:   now,
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
		existing.Letter = letter
		existing.ModuleCode = row.ModuleCode
		existing.ModuleName = row.ModuleName
		existing.ModuleOrder = row.ModuleOrder
		existing.KpOrder = row.KpOrder
		existing.SoloText = reading.Solo
		existing.WordText = reading.Word
		existing.SyncedAt = now
		existing.UpdatedAt = now
		if err := s.db.Save(&existing).Error; err != nil {
			return SyncResult{}, err
		}
		upserted++
	}
	return SyncResult{Upserted: upserted, Total: len(rows)}, nil
}

func (s *Service) List(view string) (ListResult, error) {
	if view == "" {
		view = "groups"
	}
	var assets []Asset
	if err := s.db.Order("module_order ASC, kp_order ASC, kp_id ASC").Find(&assets).Error; err != nil {
		return ListResult{}, err
	}

	dtos := make([]ItemDTO, 0, len(assets))
	for _, a := range assets {
		dtos = append(dtos, toDTO(a))
	}

	out := ListResult{View: view, Total: len(dtos)}
	if view == "table" {
		out.Items = dtos
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
				Items:       []ItemDTO{},
			}
			byMod[dto.ModuleCode] = g
			order = append(order, dto.ModuleCode)
		}
		g.Items = append(g.Items, dto)
	}
	out.Groups = make([]GroupDTO, 0, len(order))
	for _, code := range order {
		out.Groups = append(out.Groups, *byMod[code])
	}
	return out, nil
}


func glyphPublicURL(kpID int64) string {
	return fmt.Sprintf("%s/api/v1/pinyin/items/%d/glyph.png", publicBase(), kpID)
}

func (s *Service) GenerateGlyph(ctx context.Context, kpID int64) (ItemDTO, error) {
	if s.store == nil || s.renderer == nil {
		return ItemDTO{}, fmt.Errorf("字图生成未就绪（检查 MinIO 与字体）")
	}
	var asset Asset
	if err := s.db.First(&asset, "kp_id = ?", kpID).Error; err != nil {
		return ItemDTO{}, err
	}
	letter := strings.TrimSpace(asset.Letter)
	if letter == "" {
		return ItemDTO{}, fmt.Errorf("字母为空")
	}
	png, err := s.renderer.RenderPNG(letter)
	if err != nil {
		return ItemDTO{}, err
	}
	if _, err := s.store.PutBytes(ctx, storage.PinyinGlyphObjectKey(kpID), png, "image/png"); err != nil {
		return ItemDTO{}, err
	}
	asset.GlyphImageURL = glyphPublicURL(kpID)
	asset.UpdatedAt = time.Now().UTC()
	if err := s.db.Save(&asset).Error; err != nil {
		return ItemDTO{}, err
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
	if strings.TrimSpace(asset.GlyphImageURL) == "" {
		return nil, fmt.Errorf("尚未生成字图")
	}
	return s.store.GetBytes(ctx, storage.PinyinGlyphObjectKey(kpID))
}

// BatchGenerateGlyphs generates missing glyphs for one module only.
func (s *Service) BatchGenerateGlyphs(ctx context.Context, moduleCode string) (BatchResult, error) {
	moduleCode = strings.TrimSpace(moduleCode)
	if moduleCode == "" {
		return BatchResult{}, fmt.Errorf("moduleCode 不能为空")
	}
	if s.store == nil || s.renderer == nil {
		return BatchResult{}, fmt.Errorf("字图生成未就绪（检查 MinIO 与字体）")
	}
	var assets []Asset
	if err := s.db.Where("module_code = ? AND (glyph_image_url = '' OR glyph_image_url IS NULL)", moduleCode).
		Order("module_order ASC, kp_order ASC, kp_id ASC").
		Find(&assets).Error; err != nil {
		return BatchResult{}, err
	}
	out := BatchResult{}
	for _, asset := range assets {
		if _, err := s.GenerateGlyph(ctx, asset.KpID); err != nil {
			out.Failed++
			if len(out.Errors) < 10 {
				out.Errors = append(out.Errors, fmt.Sprintf("%s(%d): %v", asset.Letter, asset.KpID, err))
			}
			continue
		}
		out.Generated++
	}
	return out, nil
}

func toDTO(a Asset) ItemDTO {
	return ItemDTO{
		KpID:          a.KpID,
		Letter:        a.Letter,
		ModuleCode:    a.ModuleCode,
		ModuleName:    a.ModuleName,
		ModuleOrder:   a.ModuleOrder,
		KpOrder:       a.KpOrder,
		SoloText:      a.SoloText,
		WordText:      a.WordText,
		SoloSpeechURL: withCacheBust(a.SoloSpeechURL, a.UpdatedAt),
		WordSpeechURL: withCacheBust(a.WordSpeechURL, a.UpdatedAt),
		GlyphImageURL: withCacheBust(a.GlyphImageURL, a.UpdatedAt),
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
