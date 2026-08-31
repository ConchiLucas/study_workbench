package literacy_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/conchi/study-content-admin/internal/configclient"
	"github.com/conchi/study-content-admin/internal/db"
	"github.com/conchi/study-content-admin/internal/literacy"
)

func setupLiteracyDB(t *testing.T) *literacy.Service {
	t.Helper()
	gdb, err := db.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, gdb.Exec(`
CREATE TABLE subjects (id INTEGER PRIMARY KEY, code TEXT, name TEXT, icon TEXT, order_no INT);
CREATE TABLE modules (id INTEGER PRIMARY KEY, subject_id INT, code TEXT, name TEXT, order_no INT);
CREATE TABLE knowledge_points (id INTEGER PRIMARY KEY, module_id INT, code TEXT, title TEXT, payload TEXT, difficulty INT, order_no INT);
INSERT INTO subjects(id, code, name, icon, order_no) VALUES (1, 'literacy', '识字', '', 1);
INSERT INTO modules(id, subject_id, code, name, order_no) VALUES (1, 1, 'g2', '第2组', 2);
INSERT INTO knowledge_points(id, module_id, code, title, payload, difficulty, order_no) VALUES
 (10, 1, 'l2-1', '人', '{}', 1, 1),
 (11, 1, 'l2-2', '口', '{}', 1, 2),
 (12, 1, 'l2-3', '的', '{}', 1, 3);
`).Error)
	require.NoError(t, db.Migrate(gdb))
	return literacy.NewService(gdb, nil, nil, nil, nil, nil, nil)
}

func TestSyncAndOverride(t *testing.T) {
	svc := setupLiteracyDB(t)
	res, err := svc.Sync()
	require.NoError(t, err)
	require.Equal(t, 3, res.Total)

	list, err := svc.List("groups", nil)
	require.NoError(t, err)
	require.Equal(t, 3, list.Total)
	require.Len(t, list.Groups, 1)
	require.Equal(t, "第2组", list.Groups[0].ModuleName)

	byChar := map[string]literacy.CharDTO{}
	for _, c := range list.Groups[0].Chars {
		byChar[c.CharText] = c
	}
	require.True(t, byChar["人"].EffectiveNeedsSenseImage)
	require.False(t, byChar["的"].EffectiveNeedsSenseImage)

	needTrue := true
	filtered, err := svc.List("table", &needTrue)
	require.NoError(t, err)
	require.Equal(t, 2, filtered.Total) // 人、口
	got := map[string]bool{}
	for _, c := range filtered.Chars {
		got[c.CharText] = true
	}
	require.True(t, got["人"])
	require.True(t, got["口"])
	require.False(t, got["的"])
}

type memStore struct {
	objects map[string][]byte
	puts    int
	gets    int
}

func (m *memStore) PutPNG(ctx context.Context, objectKey string, png []byte) (string, error) {
	return m.PutBytes(ctx, objectKey, png, "image/png")
}
func (m *memStore) GetPNG(ctx context.Context, relativeOrFullKey string) ([]byte, error) {
	return m.GetBytes(ctx, relativeOrFullKey)
}
func (m *memStore) PutBytes(ctx context.Context, objectKey string, data []byte, contentType string) (string, error) {
	m.puts++
	if m.objects == nil {
		m.objects = map[string][]byte{}
	}
	cp := append([]byte(nil), data...)
	m.objects[objectKey] = cp
	return objectKey, nil
}
func (m *memStore) GetBytes(ctx context.Context, relativeOrFullKey string) ([]byte, error) {
	m.gets++
	data, ok := m.objects[relativeOrFullKey]
	if !ok {
		return nil, errNotFound
	}
	return append([]byte(nil), data...), nil
}

var errNotFound = errString("object not found")

type errString string

func (e errString) Error() string { return string(e) }
func (m *memStore) GlyphKey(kpID int64) string  { return "literacy/glyphs/" + itoa(kpID) + ".png" }
func (m *memStore) SenseKey(kpID int64) string  { return "literacy/senses/" + itoa(kpID) + ".png" }
func (m *memStore) SpeechKey(kpID int64) string { return "literacy/speech/" + itoa(kpID) + ".mp3" }

func itoa(n int64) string {
	const digits = "0123456789"
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = digits[n%10]
		n /= 10
	}
	return string(b[i:])
}

type stubVoices struct{}

func (stubVoices) LoadVoiceModels(ctx context.Context) (configclient.AIConfiguration, error) {
	return configclient.AIConfiguration{
		ActiveProviderID: "p1",
		Providers: []configclient.AIProvider{{
			ID: "p1", Type: "openai-compatible", Enabled: true,
			BaseURL: "http://x", Model: "m", Capabilities: []string{"AUDIO_TTS"},
		}},
	}, nil
}

type stubSpeech struct{ calls int }

func (s *stubSpeech) Synthesize(ctx context.Context, provider configclient.AIProvider, text string) ([]byte, error) {
	s.calls++
	return []byte("mp3-" + text), nil
}

func TestSpeechMP3CacheHit(t *testing.T) {
	gdb, err := db.OpenSQLite("file:speech_cache?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, db.Migrate(gdb))
	require.NoError(t, gdb.Exec(`
INSERT INTO literacy_assets(kp_id, char_text, module_code, module_name, module_order, kp_order, needs_sense_image)
VALUES (10, '人', 'g1', '第1组', 1, 1, 1)
`).Error)

	store := &memStore{}
	speech := &stubSpeech{}
	svc := literacy.NewService(gdb, store, nil, nil, nil, stubVoices{}, speech)

	first, err := svc.SpeechMP3(context.Background(), 10)
	require.NoError(t, err)
	require.Equal(t, []byte("mp3-人"), first)
	require.Equal(t, 1, speech.calls)
	require.Equal(t, 1, store.puts)

	second, err := svc.SpeechMP3(context.Background(), 10)
	require.NoError(t, err)
	require.Equal(t, first, second)
	require.Equal(t, 1, speech.calls, "second play must use cache, not re-synthesize")
	require.Equal(t, 1, store.puts)
	require.GreaterOrEqual(t, store.gets, 1)

	_, err = svc.RegenerateSpeech(context.Background(), 10)
	require.NoError(t, err)
	require.Equal(t, 2, speech.calls)
	require.Equal(t, 2, store.puts)
}

func TestBatchGenerateSpeechRequiresModule(t *testing.T) {
	svc := setupLiteracyDB(t)
	_, err := svc.BatchGenerateSpeech(context.Background(), "")
	require.Error(t, err)
}
