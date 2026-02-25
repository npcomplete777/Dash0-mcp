package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	// Save and clear environment
	savedAuthToken := os.Getenv("DASH0_AUTH_TOKEN")
	savedToken := os.Getenv("DASH0_TOKEN")
	savedRegion := os.Getenv("DASH0_REGION")
	savedBaseURL := os.Getenv("DASH0_BASE_URL")
	savedDebug := os.Getenv("DASH0_DEBUG")
	defer func() {
		os.Setenv("DASH0_AUTH_TOKEN", savedAuthToken)
		os.Setenv("DASH0_TOKEN", savedToken)
		os.Setenv("DASH0_REGION", savedRegion)
		os.Setenv("DASH0_BASE_URL", savedBaseURL)
		os.Setenv("DASH0_DEBUG", savedDebug)
	}()

	tests := []struct {
		name            string
		envVars         map[string]string
		wantAuthToken   string
		wantRegion      Region
		wantBaseURL     string
		wantDebug       bool
	}{
		{
			name: "defaults to EU West 1",
			envVars: map[string]string{
				"DASH0_AUTH_TOKEN": "test-token",
				"DASH0_REGION":     "",
				"DASH0_BASE_URL":   "",
				"DASH0_DEBUG":      "",
			},
			wantAuthToken: "test-token",
			wantRegion:    RegionEUWest1,
			wantBaseURL:   "https://api.eu-west-1.aws.dash0.com",
			wantDebug:     false,
		},
		{
			name: "US East 1 region",
			envVars: map[string]string{
				"DASH0_AUTH_TOKEN": "test-token",
				"DASH0_REGION":     "us-east-1",
				"DASH0_BASE_URL":   "",
				"DASH0_DEBUG":      "",
			},
			wantAuthToken: "test-token",
			wantRegion:    RegionUSEast1,
			wantBaseURL:   "https://api.us-east-1.aws.dash0.com",
			wantDebug:     false,
		},
		{
			name: "US West 2 region",
			envVars: map[string]string{
				"DASH0_AUTH_TOKEN": "test-token",
				"DASH0_REGION":     "us-west-2",
				"DASH0_BASE_URL":   "",
				"DASH0_DEBUG":      "",
			},
			wantAuthToken: "test-token",
			wantRegion:    RegionUSWest2,
			wantBaseURL:   "https://api.us-west-2.aws.dash0.com",
			wantDebug:     false,
		},
		{
			name: "custom base URL overrides region",
			envVars: map[string]string{
				"DASH0_AUTH_TOKEN": "test-token",
				"DASH0_REGION":     "eu-west-1",
				"DASH0_BASE_URL":   "https://custom.api.com",
				"DASH0_DEBUG":      "",
			},
			wantAuthToken: "test-token",
			wantRegion:    RegionEUWest1,
			wantBaseURL:   "https://custom.api.com",
			wantDebug:     false,
		},
		{
			name: "debug mode enabled with true",
			envVars: map[string]string{
				"DASH0_AUTH_TOKEN": "test-token",
				"DASH0_REGION":     "",
				"DASH0_BASE_URL":   "",
				"DASH0_DEBUG":      "true",
			},
			wantAuthToken: "test-token",
			wantRegion:    RegionEUWest1,
			wantBaseURL:   "https://api.eu-west-1.aws.dash0.com",
			wantDebug:     true,
		},
		{
			name: "debug mode enabled with 1",
			envVars: map[string]string{
				"DASH0_AUTH_TOKEN": "test-token",
				"DASH0_REGION":     "",
				"DASH0_BASE_URL":   "",
				"DASH0_DEBUG":      "1",
			},
			wantAuthToken: "test-token",
			wantRegion:    RegionEUWest1,
			wantBaseURL:   "https://api.eu-west-1.aws.dash0.com",
			wantDebug:     true,
		},
		{
			name: "debug mode enabled with yes",
			envVars: map[string]string{
				"DASH0_AUTH_TOKEN": "test-token",
				"DASH0_REGION":     "",
				"DASH0_BASE_URL":   "",
				"DASH0_DEBUG":      "yes",
			},
			wantAuthToken: "test-token",
			wantRegion:    RegionEUWest1,
			wantBaseURL:   "https://api.eu-west-1.aws.dash0.com",
			wantDebug:     true,
		},
		{
			name: "fallback to DASH0_TOKEN if AUTH_TOKEN not set",
			envVars: map[string]string{
				"DASH0_AUTH_TOKEN": "",
				"DASH0_TOKEN":      "fallback-token",
				"DASH0_REGION":     "",
				"DASH0_BASE_URL":   "",
				"DASH0_DEBUG":      "",
			},
			wantAuthToken: "fallback-token",
			wantRegion:    RegionEUWest1,
			wantBaseURL:   "https://api.eu-west-1.aws.dash0.com",
			wantDebug:     false,
		},
		{
			name: "URL-like region with https prefix",
			envVars: map[string]string{
				"DASH0_AUTH_TOKEN": "test-token",
				"DASH0_REGION":     "https://api.eu-west-1.aws.dash0.com",
				"DASH0_BASE_URL":   "",
				"DASH0_DEBUG":      "",
			},
			wantAuthToken: "test-token",
			wantRegion:    RegionEUWest1,
			wantBaseURL:   "https://api.eu-west-1.aws.dash0.com",
			wantDebug:     false,
		},
		{
			name: "URL-like region with api prefix",
			envVars: map[string]string{
				"DASH0_AUTH_TOKEN": "test-token",
				"DASH0_REGION":     "api.us-east-1.aws.dash0.com",
				"DASH0_BASE_URL":   "",
				"DASH0_DEBUG":      "",
			},
			wantAuthToken: "test-token",
			wantRegion:    RegionUSEast1,
			wantBaseURL:   "https://api.us-east-1.aws.dash0.com",
			wantDebug:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all env vars first
			os.Unsetenv("DASH0_AUTH_TOKEN")
			os.Unsetenv("DASH0_TOKEN")
			os.Unsetenv("DASH0_REGION")
			os.Unsetenv("DASH0_BASE_URL")
			os.Unsetenv("DASH0_DEBUG")

			// Set test env vars
			for k, v := range tt.envVars {
				if v != "" {
					os.Setenv(k, v)
				}
			}

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}

			if cfg.AuthToken != tt.wantAuthToken {
				t.Errorf("AuthToken = %q, want %q", cfg.AuthToken, tt.wantAuthToken)
			}
			if cfg.Region != tt.wantRegion {
				t.Errorf("Region = %q, want %q", cfg.Region, tt.wantRegion)
			}
			if cfg.BaseURL != tt.wantBaseURL {
				t.Errorf("BaseURL = %q, want %q", cfg.BaseURL, tt.wantBaseURL)
			}
			if cfg.Debug != tt.wantDebug {
				t.Errorf("Debug = %v, want %v", cfg.Debug, tt.wantDebug)
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: &Config{
				AuthToken: "test-token",
				BaseURL:   "https://api.eu-west-1.aws.dash0.com",
				Region:    RegionEUWest1,
			},
			wantErr: false,
		},
		{
			name: "missing auth token",
			config: &Config{
				AuthToken: "",
				BaseURL:   "https://api.eu-west-1.aws.dash0.com",
				Region:    RegionEUWest1,
			},
			wantErr: true,
			errMsg:  "DASH0_AUTH_TOKEN is required",
		},
		{
			name: "missing base URL",
			config: &Config{
				AuthToken: "test-token",
				BaseURL:   "",
				Region:    RegionEUWest1,
			},
			wantErr: true,
			errMsg:  "unable to determine base URL",
		},
		{
			name: "non-HTTPS base URL",
			config: &Config{
				AuthToken: "test-token",
				BaseURL:   "http://api.dash0.com",
				Region:    RegionEUWest1,
			},
			wantErr: true,
			errMsg:  "base URL must use HTTPS",
		},
		{
			name: "custom region with base URL is valid",
			config: &Config{
				AuthToken: "test-token",
				BaseURL:   "https://custom.api.com",
				Region:    "custom-region",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" {
				if err == nil || !contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %v, want error containing %q", err, tt.errMsg)
				}
			}
		})
	}
}

func TestDeriveBaseURL(t *testing.T) {
	tests := []struct {
		name   string
		region Region
		want   string
	}{
		{
			name:   "EU West 1",
			region: RegionEUWest1,
			want:   "https://api.eu-west-1.aws.dash0.com",
		},
		{
			name:   "US East 1",
			region: RegionUSEast1,
			want:   "https://api.us-east-1.aws.dash0.com",
		},
		{
			name:   "US West 2",
			region: RegionUSWest2,
			want:   "https://api.us-west-2.aws.dash0.com",
		},
		{
			name:   "unknown region",
			region: "unknown",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Region: tt.region}
			got := cfg.deriveBaseURL()
			if got != tt.want {
				t.Errorf("deriveBaseURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCoalesce(t *testing.T) {
	tests := []struct {
		name   string
		values []string
		want   string
	}{
		{
			name:   "first non-empty",
			values: []string{"first", "second"},
			want:   "first",
		},
		{
			name:   "skip empty",
			values: []string{"", "second"},
			want:   "second",
		},
		{
			name:   "all empty",
			values: []string{"", ""},
			want:   "",
		},
		{
			name:   "no values",
			values: []string{},
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := coalesce(tt.values...)
			if got != tt.want {
				t.Errorf("coalesce() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseBool(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "true", input: "true", want: true},
		{name: "TRUE", input: "TRUE", want: true},
		{name: "True", input: "True", want: true},
		{name: "1", input: "1", want: true},
		{name: "yes", input: "yes", want: true},
		{name: "YES", input: "YES", want: true},
		{name: "false", input: "false", want: false},
		{name: "0", input: "0", want: false},
		{name: "no", input: "no", want: false},
		{name: "empty", input: "", want: false},
		{name: "random", input: "random", want: false},
		{name: "with spaces", input: "  true  ", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseBool(tt.input)
			if got != tt.want {
				t.Errorf("parseBool(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestLoad_Dataset(t *testing.T) {
	// Save and clear environment
	savedDataset := os.Getenv("DASH0_DATASET")
	savedAuthToken := os.Getenv("DASH0_AUTH_TOKEN")
	defer func() {
		os.Setenv("DASH0_DATASET", savedDataset)
		os.Setenv("DASH0_AUTH_TOKEN", savedAuthToken)
	}()

	tests := []struct {
		name        string
		dataset     string
		wantDataset string
	}{
		{
			name:        "dataset configured",
			dataset:     "otel-demo-gitops",
			wantDataset: "otel-demo-gitops",
		},
		{
			name:        "dataset not configured",
			dataset:     "",
			wantDataset: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("DASH0_AUTH_TOKEN", "test-token")
			if tt.dataset != "" {
				os.Setenv("DASH0_DATASET", tt.dataset)
			} else {
				os.Unsetenv("DASH0_DATASET")
			}

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}

			if cfg.Dataset != tt.wantDataset {
				t.Errorf("Dataset = %q, want %q", cfg.Dataset, tt.wantDataset)
			}
		})
	}
}

// contains checks if substr is in s
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
