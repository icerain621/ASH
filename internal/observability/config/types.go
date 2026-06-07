package config

const Version = "ash.obs/v0.1"

// Document is the top-level ash.obs/v0.1 configuration.
type Document struct {
	Version   string           `yaml:"version" json:"version"`
	Redaction RedactionConfig  `yaml:"redaction,omitempty" json:"redaction,omitempty"`
	Sampling  SamplingConfig   `yaml:"sampling,omitempty" json:"sampling,omitempty"`
	Export    ExportConfig     `yaml:"export,omitempty" json:"export,omitempty"`
	Plugins   PluginsConfig    `yaml:"plugins,omitempty" json:"plugins,omitempty"`
}

type RedactionConfig struct {
	Enabled  bool     `yaml:"enabled" json:"enabled"`
	DenyKeys []string `yaml:"denyKeys,omitempty" json:"denyKeys,omitempty"`
	MaskValue string  `yaml:"maskValue,omitempty" json:"maskValue,omitempty"`
}

type SamplingConfig struct {
	EventExportRate float64 `yaml:"eventExportRate,omitempty" json:"eventExportRate,omitempty"`
}

type ExportConfig struct {
	AllowOutbound bool `yaml:"allowOutbound" json:"allowOutbound"`
}

type PluginsConfig struct {
	Prometheus *PrometheusPlugin `yaml:"prometheus,omitempty" json:"prometheus,omitempty"`
	Otel       *OtelPlugin       `yaml:"otel,omitempty" json:"otel,omitempty"`
	Console    *ConsolePlugin    `yaml:"console,omitempty" json:"console,omitempty"`
}

type PrometheusPlugin struct {
	Enabled bool   `yaml:"enabled" json:"enabled"`
	Path    string `yaml:"path,omitempty" json:"path,omitempty"`
}

type OtelPlugin struct {
	Enabled  bool   `yaml:"enabled" json:"enabled"`
	Endpoint string `yaml:"endpoint,omitempty" json:"endpoint,omitempty"`
	Insecure bool   `yaml:"insecure,omitempty" json:"insecure,omitempty"`
}

type ConsolePlugin struct {
	Enabled bool   `yaml:"enabled" json:"enabled"`
	Level   string `yaml:"level,omitempty" json:"level,omitempty"`
}

// ValidationIssue describes a config validation error.
type ValidationIssue struct {
	Path    string `json:"path"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ValidationResult is the outcome of ParseAndValidate.
type ValidationResult struct {
	OK     bool              `json:"ok"`
	Doc    *Document         `json:"doc,omitempty"`
	Issues []ValidationIssue `json:"issues,omitempty"`
}
