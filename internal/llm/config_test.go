package llm

import (
	"strings"
	"testing"
)

func TestLoadConfigFromEnvDoesNotReadValuesWhenDisabled(t *testing.T) {
	setValidLLMEnv(t)

	got, err := LoadConfigFromEnv(false)
	if err != nil {
		t.Fatalf("LoadConfigFromEnv returned error: %v", err)
	}

	want := Config{}
	if got != want {
		t.Fatalf("Config = %#v, want %#v", got, want)
	}
}

func TestLoadConfigFromEnvLoadsValidConfigWhenEnabled(t *testing.T) {
	setValidLLMEnv(t)

	got, err := LoadConfigFromEnv(true)
	if err != nil {
		t.Fatalf("LoadConfigFromEnv returned error: %v", err)
	}

	want := Config{
		Enabled:  true,
		Provider: ProviderOpenAICompatible,
		BaseURL:  "https://api.openai.com/v1",
		Model:    "gpt-4.1-mini",
		APIKey:   "sk-test-secret",
	}
	if got != want {
		t.Fatalf("Config = %#v, want %#v", got, want)
	}
}

func TestLoadConfigFromEnvWithOverridesMergesNonSecretSettings(t *testing.T) {
	setValidLLMEnv(t)

	got, err := LoadConfigFromEnvWithOverrides(true, ConfigOverrides{
		Provider: "  openai-compatible  ",
		BaseURL:  "  https://provider.example/v1  ",
		Model:    "  override-model  ",
	})
	if err != nil {
		t.Fatalf("LoadConfigFromEnvWithOverrides returned error: %v", err)
	}

	want := Config{
		Enabled:  true,
		Provider: ProviderOpenAICompatible,
		BaseURL:  "https://provider.example/v1",
		Model:    "override-model",
		APIKey:   "sk-test-secret",
	}
	if got != want {
		t.Fatalf("Config = %#v, want %#v", got, want)
	}
}

func TestLoadConfigFromEnvWithOverridesFallsBackForEmptyOverrides(t *testing.T) {
	setValidLLMEnv(t)

	got, err := LoadConfigFromEnvWithOverrides(true, ConfigOverrides{
		Provider: "  ",
		BaseURL:  "https://provider.example/v1",
	})
	if err != nil {
		t.Fatalf("LoadConfigFromEnvWithOverrides returned error: %v", err)
	}

	if got.Provider != ProviderOpenAICompatible {
		t.Fatalf("Provider = %q, want environment value %q", got.Provider, ProviderOpenAICompatible)
	}
	if got.BaseURL != "https://provider.example/v1" {
		t.Fatalf("BaseURL = %q, want override value", got.BaseURL)
	}
	if got.Model != "gpt-4.1-mini" {
		t.Fatalf("Model = %q, want environment value", got.Model)
	}
	if got.APIKey != "sk-test-secret" {
		t.Fatalf("APIKey = %q, want environment value", got.APIKey)
	}
}

func TestLoadConfigFromEnvWithOverridesValidatesAfterMerge(t *testing.T) {
	clearLLMEnv(t)
	t.Setenv(EnvAPIKey, "sk-env-only-secret")

	got, err := LoadConfigFromEnvWithOverrides(true, ConfigOverrides{
		Provider: ProviderOpenAICompatible,
		BaseURL:  "https://provider.example/v1",
		Model:    "override-model",
	})
	if err != nil {
		t.Fatalf("LoadConfigFromEnvWithOverrides returned error: %v", err)
	}

	if got.Provider != ProviderOpenAICompatible || got.BaseURL != "https://provider.example/v1" || got.Model != "override-model" {
		t.Fatalf("merged config = %#v, want CLI-safe overrides", got)
	}
	if got.APIKey != "sk-env-only-secret" {
		t.Fatalf("APIKey = %q, want environment value", got.APIKey)
	}
}

func TestLoadConfigFromEnvWithOverridesStillRequiresEnvironmentAPIKey(t *testing.T) {
	clearLLMEnv(t)

	_, err := LoadConfigFromEnvWithOverrides(true, ConfigOverrides{
		Provider: ProviderOpenAICompatible,
		BaseURL:  "https://provider.example/v1",
		Model:    "override-model",
	})
	if err == nil {
		t.Fatal("LoadConfigFromEnvWithOverrides returned nil error")
	}
	if !strings.Contains(err.Error(), EnvAPIKey) {
		t.Fatalf("error = %q, want message containing %q", err.Error(), EnvAPIKey)
	}
	for _, setting := range []string{EnvProvider, EnvBaseURL, EnvModel} {
		if strings.Contains(err.Error(), setting) {
			t.Fatalf("error = %q, did not expect merged setting %q to be missing", err.Error(), setting)
		}
	}
}

func TestLoadConfigFromEnvWithOverridesDoesNothingWhenDisabled(t *testing.T) {
	setValidLLMEnv(t)

	got, err := LoadConfigFromEnvWithOverrides(false, ConfigOverrides{
		Provider: "unsupported-provider",
		BaseURL:  "://invalid",
		Model:    "override-model",
	})
	if err != nil {
		t.Fatalf("LoadConfigFromEnvWithOverrides returned error: %v", err)
	}
	if got != (Config{}) {
		t.Fatalf("Config = %#v, want zero value", got)
	}
}

func TestLoadConfigFromEnvRejectsMissingRequiredValuesWhenEnabled(t *testing.T) {
	clearLLMEnv(t)

	_, err := LoadConfigFromEnv(true)
	if err == nil {
		t.Fatal("LoadConfigFromEnv returned nil error")
	}

	for _, want := range []string{EnvProvider, EnvBaseURL, EnvModel, EnvAPIKey} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %q, want message containing %q", err.Error(), want)
		}
	}
}

func TestLoadConfigFromEnvRejectsMissingAPIKeyWhenEnabled(t *testing.T) {
	setValidLLMEnv(t)
	t.Setenv(EnvAPIKey, "")

	_, err := LoadConfigFromEnv(true)
	if err == nil {
		t.Fatal("LoadConfigFromEnv returned nil error")
	}
	if !strings.Contains(err.Error(), EnvAPIKey) {
		t.Fatalf("error = %q, want message containing %q", err.Error(), EnvAPIKey)
	}
}

func TestLoadConfigFromEnvRejectsUnsupportedProvider(t *testing.T) {
	setValidLLMEnv(t)
	t.Setenv(EnvProvider, "example-provider")

	_, err := LoadConfigFromEnv(true)
	if err == nil {
		t.Fatal("LoadConfigFromEnv returned nil error")
	}

	for _, want := range []string{EnvProvider, ProviderOpenAICompatible} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %q, want message containing %q", err.Error(), want)
		}
	}
}

func TestLoadConfigFromEnvRejectsInvalidBaseURL(t *testing.T) {
	setValidLLMEnv(t)
	t.Setenv(EnvBaseURL, "api.openai.com/v1")

	_, err := LoadConfigFromEnv(true)
	if err == nil {
		t.Fatal("LoadConfigFromEnv returned nil error")
	}
	if !strings.Contains(err.Error(), EnvBaseURL) {
		t.Fatalf("error = %q, want message containing %q", err.Error(), EnvBaseURL)
	}
}

func TestLoadConfigFromEnvDoesNotLeakAPIKeyInErrors(t *testing.T) {
	const secret = "sk-secret-value-that-must-not-appear"
	setValidLLMEnv(t)
	t.Setenv(EnvAPIKey, secret)
	t.Setenv(EnvBaseURL, "api.openai.com/v1")

	_, err := LoadConfigFromEnv(true)
	if err == nil {
		t.Fatal("LoadConfigFromEnv returned nil error")
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("error leaked API key: %q", err.Error())
	}
}

func setValidLLMEnv(t *testing.T) {
	t.Helper()

	t.Setenv(EnvProvider, ProviderOpenAICompatible)
	t.Setenv(EnvBaseURL, "https://api.openai.com/v1")
	t.Setenv(EnvModel, "gpt-4.1-mini")
	t.Setenv(EnvAPIKey, "sk-test-secret")
}

func clearLLMEnv(t *testing.T) {
	t.Helper()

	t.Setenv(EnvProvider, "")
	t.Setenv(EnvBaseURL, "")
	t.Setenv(EnvModel, "")
	t.Setenv(EnvAPIKey, "")
}
