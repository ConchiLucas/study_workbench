package system

import (
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/conchi/go-react-template/server/config"
	sysModel "github.com/conchi/go-react-template/server/model/system"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func openAIConfigServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := db.AutoMigrate(&sysModel.AIProviderConfig{}, &sysModel.SentenceExecutorConfig{}); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	return db
}

func legacyAIConfig(active string, ids ...string) config.AI {
	providers := make(map[string]config.AIProvider, len(ids))
	for _, id := range ids {
		providers[id] = config.AIProvider{
			Label: id, Type: config.AIProviderTypeOpenAICompatible,
			BaseURL: "https://" + id + ".example/v1", ApiKey: id + "-secret",
			Model: id + "-model", MaxTokens: 4096,
		}
	}
	return config.AI{Active: active, Providers: providers}
}

func TestAIConfigServiceSaveProtectsCLISentenceExecutor(t *testing.T) {
	db := openAIConfigServiceTestDB(t)
	target := sysModel.SentenceExecutorConfig{SingletonKey: "default", ExecutorType: "cli", ExecutorID: "codex-local"}
	if err := db.Create(&target).Error; err != nil {
		t.Fatalf("seed target: %v", err)
	}

	if err := new(AIConfigService).SaveConfig(db, legacyAIConfig("alpha", "alpha", "beta")); err != nil {
		t.Fatalf("save legacy AI config: %v", err)
	}
	var activeCount int64
	if err := db.Model(&sysModel.AIProviderConfig{}).Where("active = ?", true).Count(&activeCount).Error; err != nil {
		t.Fatalf("count active providers: %v", err)
	}
	if activeCount != 0 {
		t.Fatalf("CLI target allowed %d legacy API providers to become active", activeCount)
	}
	var persistedTarget sysModel.SentenceExecutorConfig
	if err := db.First(&persistedTarget, "singleton_key = ?", "default").Error; err != nil {
		t.Fatalf("reload target: %v", err)
	}
	if persistedTarget.ExecutorType != "cli" || persistedTarget.ExecutorID != "codex-local" {
		t.Fatalf("legacy save changed singleton: %#v", persistedTarget)
	}
}

func TestAIConfigServiceSaveRejectsMissingProtectedAPITargetAndRollsBack(t *testing.T) {
	db := openAIConfigServiceTestDB(t)
	seed := sysModel.AIProviderConfig{
		ProviderID: "protected", Label: "Protected", Type: config.AIProviderTypeOpenAICompatible,
		BaseURL: "https://protected.example/v1", ApiKey: "stored-secret", Model: "stored-model", Active: true,
	}
	if err := db.Create(&seed).Error; err != nil {
		t.Fatalf("seed provider: %v", err)
	}
	target := sysModel.SentenceExecutorConfig{SingletonKey: "default", ExecutorType: "api", ExecutorID: "protected"}
	if err := db.Create(&target).Error; err != nil {
		t.Fatalf("seed target: %v", err)
	}

	err := new(AIConfigService).SaveConfig(db, legacyAIConfig("replacement", "replacement"))
	if err == nil || !strings.Contains(err.Error(), "protected") || !strings.Contains(err.Error(), "不能删除") {
		t.Fatalf("expected protected target error, got %v", err)
	}
	var rows []sysModel.AIProviderConfig
	if err := db.Find(&rows).Error; err != nil {
		t.Fatalf("reload providers: %v", err)
	}
	if len(rows) != 1 || rows[0].ProviderID != "protected" || rows[0].ApiKey != "stored-secret" || !rows[0].Active {
		t.Fatalf("legacy save was not rolled back: %#v", rows)
	}
}

func TestAIConfigServiceSaveKeepsProtectedAPITargetActive(t *testing.T) {
	db := openAIConfigServiceTestDB(t)
	target := sysModel.SentenceExecutorConfig{SingletonKey: "default", ExecutorType: "api", ExecutorID: "protected"}
	if err := db.Create(&target).Error; err != nil {
		t.Fatalf("seed target: %v", err)
	}

	if err := new(AIConfigService).SaveConfig(db, legacyAIConfig("other", "protected", "other")); err != nil {
		t.Fatalf("save legacy AI config: %v", err)
	}
	var rows []sysModel.AIProviderConfig
	if err := db.Order("provider_id").Find(&rows).Error; err != nil {
		t.Fatalf("reload providers: %v", err)
	}
	if len(rows) != 2 || rows[0].ProviderID != "other" || rows[0].Active || rows[1].ProviderID != "protected" || !rows[1].Active {
		t.Fatalf("protected API target was not the sole active provider: %#v", rows)
	}
}

func TestAIConfigServiceSaveWithoutSingletonKeepsLegacyBehavior(t *testing.T) {
	db := openAIConfigServiceTestDB(t)
	if err := new(AIConfigService).SaveConfig(db, legacyAIConfig("beta", "alpha", "beta")); err != nil {
		t.Fatalf("save legacy AI config: %v", err)
	}
	var rows []sysModel.AIProviderConfig
	if err := db.Order("provider_id").Find(&rows).Error; err != nil {
		t.Fatalf("reload providers: %v", err)
	}
	if len(rows) != 2 || rows[0].Active || !rows[1].Active {
		t.Fatalf("legacy active behavior changed without singleton: %#v", rows)
	}
}

func TestAIConfigServiceLoadRespectsSentenceExecutorSingleton(t *testing.T) {
	tests := []struct {
		name       string
		targetType string
		targetID   string
		wantActive string
	}{
		{name: "cli leaves API inactive", targetType: "cli", targetID: "codex-local", wantActive: ""},
		{name: "api selects protected provider", targetType: "api", targetID: "beta", wantActive: "beta"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db := openAIConfigServiceTestDB(t)
			rows := []sysModel.AIProviderConfig{
				{ProviderID: "alpha", Label: "Alpha", Type: config.AIProviderTypeOpenAICompatible, BaseURL: "https://alpha.example/v1", ApiKey: "alpha", Model: "alpha", Active: true},
				{ProviderID: "beta", Label: "Beta", Type: config.AIProviderTypeOpenAICompatible, BaseURL: "https://beta.example/v1", ApiKey: "beta", Model: "beta", Active: false},
			}
			if err := db.Create(&rows).Error; err != nil {
				t.Fatalf("seed providers: %v", err)
			}
			if err := db.Create(&sysModel.SentenceExecutorConfig{SingletonKey: "default", ExecutorType: test.targetType, ExecutorID: test.targetID}).Error; err != nil {
				t.Fatalf("seed target: %v", err)
			}

			loaded, found, err := new(AIConfigService).LoadConfig(db)
			if err != nil {
				t.Fatalf("load config: %v", err)
			}
			if !found || loaded.Active != test.wantActive {
				t.Fatalf("expected active %q, got found=%v config=%#v", test.wantActive, found, loaded)
			}
		})
	}
}

func TestAIConfigServiceLoadFallsBackEmptyLabelToProviderID(t *testing.T) {
	db := openAIConfigServiceTestDB(t)
	if err := db.Create(&sysModel.AIProviderConfig{
		ProviderID: "legacy", Type: config.AIProviderTypeOpenAICompatible,
		BaseURL: "https://legacy.example/v1", ApiKey: "secret", Model: "model", Active: true,
	}).Error; err != nil {
		t.Fatalf("seed provider: %v", err)
	}

	loaded, found, err := new(AIConfigService).LoadConfig(db)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if !found || loaded.Providers["legacy"].Label != "legacy" {
		t.Fatalf("empty legacy label did not fall back to ID: %#v", loaded)
	}
}

func TestAIConfigServiceSaveAndPublishSerializesCommittedRuntime(t *testing.T) {
	db := openAIConfigServiceTestDB(t)
	service := new(AIConfigService)
	publishAStarted := make(chan struct{})
	releasePublishA := make(chan struct{})
	publishBFinished := make(chan struct{})
	writerBStarted := make(chan struct{})
	type result struct {
		config config.AI
		err    error
	}
	resultA := make(chan result, 1)
	resultB := make(chan result, 1)
	var runtimeMu sync.Mutex
	var runtime config.AI

	go func() {
		effective, err := service.SaveConfigAndPublish(db, legacyAIConfig("alpha", "alpha"), func(committed config.AI) {
			close(publishAStarted)
			<-releasePublishA
			runtimeMu.Lock()
			runtime = committed
			runtimeMu.Unlock()
		})
		resultA <- result{config: effective, err: err}
	}()

	select {
	case <-publishAStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("writer A did not reach publication")
	}
	go func() {
		close(writerBStarted)
		effective, err := service.SaveConfigAndPublish(db, legacyAIConfig("beta", "beta"), func(committed config.AI) {
			runtimeMu.Lock()
			runtime = committed
			runtimeMu.Unlock()
			close(publishBFinished)
		})
		resultB <- result{config: effective, err: err}
	}()
	<-writerBStarted
	select {
	case <-publishBFinished:
		t.Fatal("writer B published inside writer A's commit-to-publish window")
	case <-time.After(100 * time.Millisecond):
	}
	close(releasePublishA)

	if result := <-resultA; result.err != nil || result.config.Active != "alpha" {
		t.Fatalf("writer A failed: config=%#v err=%v", result.config, result.err)
	}
	if result := <-resultB; result.err != nil || result.config.Active != "beta" {
		t.Fatalf("writer B failed: config=%#v err=%v", result.config, result.err)
	}
	runtimeMu.Lock()
	runtimeSnapshot := runtime
	runtimeMu.Unlock()
	if runtimeSnapshot.Active != "beta" {
		t.Fatalf("runtime did not retain latest writer: %#v", runtimeSnapshot)
	}
	loaded, found, err := service.LoadConfig(db)
	if err != nil || !found || loaded.Active != "beta" {
		t.Fatalf("database did not retain latest writer: found=%v config=%#v err=%v", found, loaded, err)
	}
}

func TestAIConfigPublicationBlocksUnifiedWriterUntilPublished(t *testing.T) {
	db := openExecutionConfigTestDB(t)
	publishStarted := make(chan struct{})
	releasePublish := make(chan struct{})
	unifiedAtCoordinator := make(chan struct{})
	unifiedDone := make(chan error, 1)
	var coordinatorCalls int32
	executionConfigCoordinatorHook = func() {
		if atomic.AddInt32(&coordinatorCalls, 1) == 2 {
			close(unifiedAtCoordinator)
		}
	}
	t.Cleanup(func() { executionConfigCoordinatorHook = nil })

	legacyDone := make(chan error, 1)
	go func() {
		_, err := new(AIConfigService).SaveConfigAndPublish(db, legacyAIConfig("alpha", "alpha"), func(config.AI) {
			close(publishStarted)
			<-releasePublish
		})
		legacyDone <- err
	}()
	select {
	case <-publishStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("legacy writer did not reach publication")
	}

	input := validExecutionConfigInput()
	input.ActiveTarget.ID = "beta"
	input.APIProviders[0].ID = "beta"
	input.APIProviders[0].APIKey = "beta-secret"
	go func() { unifiedDone <- new(ExecutionConfigService).Save(db, input) }()
	select {
	case <-unifiedAtCoordinator:
	case <-time.After(5 * time.Second):
		t.Fatal("unified writer did not reach coordinator")
	}
	select {
	case err := <-unifiedDone:
		t.Fatalf("unified writer crossed publication window: %v", err)
	default:
	}

	close(releasePublish)
	if err := <-legacyDone; err != nil {
		t.Fatalf("legacy writer: %v", err)
	}
	if err := <-unifiedDone; err != nil {
		t.Fatalf("unified writer: %v", err)
	}
	loaded, err := new(ExecutionConfigService).Load(db)
	if err != nil {
		t.Fatalf("load final execution config: %v", err)
	}
	if loaded.ActiveTarget != (ExecutionTarget{Type: "api", ID: "beta"}) {
		t.Fatalf("unified writer was not latest: %#v", loaded.ActiveTarget)
	}
}

func TestAIConfigServiceReadSnapshotDeepCopiesFallback(t *testing.T) {
	fallback := config.AI{Active: "fallback", Providers: map[string]config.AIProvider{
		"fallback": {Label: "Fallback", BaseURL: "https://fallback.example/v1", ApiKey: "secret", Model: "model"},
	}}
	snapshot, fromDatabase, err := new(AIConfigService).ReadConfigSnapshot(nil, func() config.AI {
		return fallback
	}, nil)
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	if fromDatabase {
		t.Fatal("fallback snapshot reported database source")
	}
	fallback.Providers["fallback"] = config.AIProvider{Label: "Mutated"}
	if snapshot.Providers["fallback"].Label != "Fallback" {
		t.Fatalf("snapshot shared fallback map: %#v", snapshot)
	}
}
