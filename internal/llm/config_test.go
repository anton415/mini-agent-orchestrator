package llm

import (
	"strings"
	"testing"
)

func TestLoadConfigFromEnvAllowsMissingValuesWhenDisabled(t *testing.T) {
	clearLLMEnv(t)

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
