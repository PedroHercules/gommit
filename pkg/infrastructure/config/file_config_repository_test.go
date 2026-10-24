package config

import (
	"os"
	"path/filepath"
	"testing"
)

func testFileConfigRepository(t *testing.T) *FileConfigRepository {
	t.Helper()
	dir := t.TempDir()
	return &FileConfigRepository{
		configDir:  dir,
		configFile: filepath.Join(dir, "config.json"),
		keyring:    NewSimpleKeyringService(dir),
	}
}

func TestProviderCredentialsAndModelsAreStoredSeparately(t *testing.T) {
	repo := testFileConfigRepository(t)
	if err := repo.SaveProviderAPIKey("openrouter", "sk-or-v1-openrouter-key-123456"); err != nil {
		t.Fatal(err)
	}
	if err := repo.SaveProviderAPIKey("grok", "xai-grok-key-123456789"); err != nil {
		t.Fatal(err)
	}
	if err := repo.SaveProviderDefaultModel("openrouter", "vendor/model"); err != nil {
		t.Fatal(err)
	}
	if err := repo.SaveProviderDefaultModel("grok", "grok-4.7"); err != nil {
		t.Fatal(err)
	}
	if err := repo.SaveProviderAuthMethod("grok", "oauth"); err != nil {
		t.Fatal(err)
	}
	if err := repo.SaveActiveProvider("grok"); err != nil {
		t.Fatal(err)
	}

	config, err := repo.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if config.Provider != "grok" || config.AuthMethod != "oauth" || config.APIKey != "xai-grok-key-123456789" || config.DefaultModel != "grok-4.7" {
		t.Fatalf("unexpected active config: %+v", config)
	}
	openRouterModel, err := repo.LoadProviderDefaultModel("openrouter")
	if err != nil || openRouterModel != "vendor/model" {
		t.Fatalf("OpenRouter model = %q, error %v", openRouterModel, err)
	}
	openRouterKey, err := repo.LoadProviderAPIKey("openrouter")
	if err != nil || openRouterKey != "sk-or-v1-openrouter-key-123456" {
		t.Fatalf("OpenRouter API key = %q, error %v", openRouterKey, err)
	}
}

func TestLegacyOpenRouterConfigurationRemainsReadable(t *testing.T) {
	repo := testFileConfigRepository(t)
	if err := os.WriteFile(repo.configFile, []byte(`{"default_model":"vendor/legacy-model","version":"1.0"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := repo.keyring.Set("gmit", "api_key", "sk-or-v1-legacy-key-123456"); err != nil {
		t.Fatal(err)
	}

	config, err := repo.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if config.Provider != "openrouter" || config.DefaultModel != "vendor/legacy-model" || config.APIKey != "sk-or-v1-legacy-key-123456" {
		t.Fatalf("legacy configuration was not preserved: %+v", config)
	}
	enabled, err := repo.LoadSecurityScanEnabled()
	if err != nil || enabled {
		t.Fatalf("legacy security scan setting = %v, %v; want false, nil", enabled, err)
	}
}

func TestSecurityScanSettingDefaultsOffAndPersists(t *testing.T) {
	repo := testFileConfigRepository(t)
	enabled, err := repo.LoadSecurityScanEnabled()
	if err != nil || enabled {
		t.Fatalf("initial security scan setting = %v, %v; want false, nil", enabled, err)
	}

	if err := repo.SaveSecurityScanEnabled(true); err != nil {
		t.Fatal(err)
	}
	enabled, err = repo.LoadSecurityScanEnabled()
	if err != nil || !enabled {
		t.Fatalf("enabled security scan setting = %v, %v; want true, nil", enabled, err)
	}

	if err := repo.SaveSecurityScanEnabled(false); err != nil {
		t.Fatal(err)
	}
	enabled, err = repo.LoadSecurityScanEnabled()
	if err != nil || enabled {
		t.Fatalf("disabled security scan setting = %v, %v; want false, nil", enabled, err)
	}
}
