package llm

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

const (
	ProviderOpenAICompatible = "openai-compatible"

	EnvProvider = "MAO_LLM_PROVIDER"
	EnvBaseURL  = "MAO_LLM_BASE_URL"
	EnvModel    = "MAO_LLM_MODEL"
	EnvAPIKey   = "MAO_LLM_API_KEY"
)

// Config contains the provider settings required for LLM execution.
type Config struct {
	Enabled  bool
	Provider string
	BaseURL  string
	Model    string
	APIKey   string
}

// ConfigOverrides contains the non-secret LLM settings that callers may
// override without accepting credentials through command-line arguments.
type ConfigOverrides struct {
	Provider string
	BaseURL  string
	Model    string
}

// LoadConfigFromEnv reads LLM configuration from environment variables.
func LoadConfigFromEnv(enabled bool) (Config, error) {
	return LoadConfigFromEnvWithOverrides(enabled, ConfigOverrides{})
}

// LoadConfigFromEnvWithOverrides reads LLM configuration from environment
// variables, applies non-empty non-secret overrides, and validates the merged
// result. The API key is always read from the environment.
func LoadConfigFromEnvWithOverrides(enabled bool, overrides ConfigOverrides) (Config, error) {
	if !enabled {
		return Config{}, nil
	}

	cfg := Config{
		Enabled:  true,
		Provider: strings.TrimSpace(os.Getenv(EnvProvider)),
		BaseURL:  strings.TrimSpace(os.Getenv(EnvBaseURL)),
		Model:    strings.TrimSpace(os.Getenv(EnvModel)),
		APIKey:   strings.TrimSpace(os.Getenv(EnvAPIKey)),
	}

	if value := strings.TrimSpace(overrides.Provider); value != "" {
		cfg.Provider = value
	}
	if value := strings.TrimSpace(overrides.BaseURL); value != "" {
		cfg.BaseURL = value
	}
	if value := strings.TrimSpace(overrides.Model); value != "" {
		cfg.Model = value
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// Validate checks whether the config is complete and supported for LLM execution.
func (cfg Config) Validate() error {
	var missing []string

	if cfg.Provider == "" {
		missing = append(missing, EnvProvider)
	}
	if cfg.BaseURL == "" {
		missing = append(missing, EnvBaseURL)
	}
	if cfg.Model == "" {
		missing = append(missing, EnvModel)
	}
	if cfg.APIKey == "" {
		missing = append(missing, EnvAPIKey)
	}

	if len(missing) > 0 {
		return fmt.Errorf("invalid LLM config: missing required environment variable(s): %s", strings.Join(missing, ", "))
	}

	if cfg.Provider != ProviderOpenAICompatible {
		return fmt.Errorf("invalid LLM config: %s must be %q", EnvProvider, ProviderOpenAICompatible)
	}

	parsedBaseURL, err := url.Parse(cfg.BaseURL)
	if err != nil || parsedBaseURL.Scheme == "" || parsedBaseURL.Host == "" {
		return fmt.Errorf("invalid LLM config: %s must be an absolute HTTP(S) URL", EnvBaseURL)
	}
	if parsedBaseURL.Scheme != "http" && parsedBaseURL.Scheme != "https" {
		return fmt.Errorf("invalid LLM config: %s must use http or https", EnvBaseURL)
	}

	return nil
}
