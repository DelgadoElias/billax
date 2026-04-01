package provider

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ProviderCapabilities declares what a provider can do.
// Connectors self-declare via Capabilities(); YAML config is the deployment gate.
type ProviderCapabilities struct {
	Plans     bool `yaml:"plans"`       // supports plan-based billing
	PayPerUse bool `yaml:"pay_per_use"` // supports variable/metered billing
}

// CapabilitiesConfig maps provider name → capabilities (from YAML).
type CapabilitiesConfig map[string]ProviderCapabilities

// defaultCapabilities is used when the file is absent or a provider has no entry.
var defaultCapabilities = ProviderCapabilities{
	Plans:     true,
	PayPerUse: false,
}

// LoadCapabilitiesFile reads the YAML file at path.
// If the file does not exist, returns an empty CapabilitiesConfig (defaults apply per-provider).
// Any other read or parse error is returned as a hard failure.
func LoadCapabilitiesFile(path string) (CapabilitiesConfig, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return CapabilitiesConfig{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading capabilities file %s: %w", path, err)
	}

	var cfg CapabilitiesConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing capabilities file %s: %w", path, err)
	}
	return cfg, nil
}
