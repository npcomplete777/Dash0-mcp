package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	// Save original env vars and restore after test
	originalToken := os.Getenv("DASH0_AUTH_TOKEN")
	originalRegion := os.Getenv("DASH0_REGION")
	originalBaseURL := os.Getenv("DASH0_BASE_URL")
	originalDebug := os.Getenv("DASH0_DEBUG")
	defer func() {
		os.Setenv("DASH0_AUTH_TOKEN", originalToken)
		os.Setenv("DASH0_REGION", originalRegion)
		os.Setenv("DASH0_BASE_URL", originalBaseURL)
		os.Setenv("DASH0_DEBUG", originalDebug)
	}()

	tests := []struct {
		name           string
		envVars        map[string]string
		expectedRegion Region
		expectedURL    string
		expectedDebug  bool
	}{
		{
			name: "default region when not set",
			envVars: map[string]string{
				"DASH0_AUTH_TOKEN": "test-token",
				"DASH0_REGION":     "",
				"DASH0_BASE_URL":   "",
				"DASH0_DEBUG":      "",
			},
			expectedRegion: RegionEUWest1,
			expectedURL:    "https://api.eu-west-1.aws.dash0.com",
			expectedDebug:  false,
		},
		{
			name: "eu-west-1 region",
			envVars: map[string]string{
				"DASH0_AUTH_TOKEN": "test-token",
				"DASH0_REGION":     "eu-west-1",
				"DASH0_BASE_URL":   "",
				"DASH0_DEBUG":      "",
			},
			expectedRegion: RegionEUWest1,
			expectedURL:    "https://api.eu-west-1.aws.dash0.com",
			expectedDebug:  false,
		},
		{
			name: "us-east-1 region",
			envVars: map[string]string{
				"DASH0_AUTH_TOKEN": "test-token",
				"DASH0_REGION":     "us-east-1",
				"DASH0_BASE_URL":   "",
				"DASH0_DEBUG":      "",
			},
			expectedRegion: RegionUSEast1,
			expectedURL:    "https://api.us-east-1.aws.dash0.com",
			expectedDebug:  false,
		},
		{
			name: "us-west-2 region",
			envVars: map[string]string{
				"DASH0_AUTH_TOKEN": "test-token",
				"DASH0_REGION":     "us-west-2",
				"DASH0_BASE_URL":   "",
				"DASH0_DEBUG":      "",
			},
			expectedRegion: RegionUSWest2,
			expectedURL:    "https://api.us-west-2.aws.dash0.com",
			expectedDebug:  false,
		},
		{
			name: "custom base URL overrides region",
			envVars: map[string]string{
				"DASH0_AUTH_TOKEN": "test-token",
				"DASH0_REGION":     "eu-west-1",
				"DASH0_BASE_URL":   "https://custom.dash0.example.com",
				"DASH0_DEBUG":      "",
			},
			expectedRegion: RegionEUWest1,
			expectedURL:    "https://custom.dash0.example.com",
			expectedDebug:  false,
		},
		{
			name: "debug mode enabled with true",
			envVars: map[string]string{
				"DASH0_AUTH_TOKEN": "test-token",
				"DASH0_REGION":     "",
				"DASH0_BASE_URL":   "",
				"DASH0_DEBUG":      "true",
			},
			expectedRegion: RegionEUWest1,
			expectedURL:    "https://api.eu-west-1.aws.dash0.com",
			expectedDebug:  true,
		},
		{
			name: "debug mode enabled with 1",
			envVars: map[string]string{
				"DASH0_AUTH_TOKEN": "test-token",
				"DASH0_REGION":     "",
				"DASH0_BASE_URL":   "",
				"DASH0_DEBUG":      "1",
			},
			expectedRegion: RegionEUWest1,
			expectedURL:    "https://api.eu-west-1.aws.dash0.com",
			expectedDebug:  true,
		},
		{
			name: "debug mode enabled with yes",
			envVars: map[string]string{
				"DASH0_AUTH_TOKEN": "test-token",
				"DASH0_REGION":     "",
				"DASH0_BASE_URL":   "",
				"DASH0_DEBUG":      "yes",
			},
			expectedRegion: RegionEUWest1,
			expectedURL:    "https://api.eu-west-1.aws.dash0.com",
			expectedDebug:  true,
		},
		{
			name: "URL-like region value is handled",
			envVars: map[string]string{
				"DASH0_AUTH_TOKEN": "test-token",
				"DASH0_REGION":     "api.eu-west-1.aws.dash0.com",
				"DASH0_BASE_URL":   "",
				"DASH0_DEBUG":      "",
			},
			expectedRegion: RegionEUWest1,
			expectedURL:    "https://api.eu-west-1.aws.dash0.com",
			expectedDebug:  false,
		},
		{
			name: "full URL in region is handled",
			envVars: map[string]string{
				"DASH0_AUTH_TOKEN": "test-token",
				"DASH0_REGION":     "https://api.us-east-1.aws.dash0.com",
				"DASH0_BASE_URL":   "",
				"DASH0_DEBUG":      "",
			},
			expectedRegion: RegionUSEast1,
			expectedURL:    "https://api.us-east-1.aws.dash0.com",
			expectedDebug:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() returned error: %v", err)
			}

			if cfg.Region != tt.expectedRegion {
				t.Errorf("Region = %v, want %v", cfg.Region, tt.expectedRegion)
			}

			if cfg.BaseURL != tt.expectedURL {
				t.Errorf("BaseURL = %v, want %v", cfg.BaseURL, tt.expectedURL)
			}

			if cfg.Debug != tt.expectedDebug {
				t.Errorf("Debug = %v, want %v", cfg.Debug, tt.expectedDebug)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: &Config{
				AuthToken: "test-token",
				Region:    RegionEUWest1,
				BaseURL:   "https://api.eu-west-1.aws.dash0.com",
			},
			expectError: false,
		},
		{
			name: "missing auth token",
			config: &Config{
				AuthToken: "",
				Region:    RegionEUWest1,
				BaseURL:   "https://api.eu-west-1.aws.dash0.com",
			},
			expectError: true,
			errorMsg:    "DASH0_AUTH_TOKEN is required",
		},
		{
			name: "missing base URL",
			config: &Config{
				AuthToken: "test-token",
				Region:    RegionEUWest1,
				BaseURL:   "",
			},
			expectError: true,
			errorMsg:    "unable to determine base URL",
		},
		{
			name: "non-HTTPS base URL",
			config: &Config{
				AuthToken: "test-token",
				Region:    RegionEUWest1,
				BaseURL:   "http://api.eu-west-1.aws.dash0.com",
			},
			expectError: true,
			errorMsg:    "base URL must use HTTPS",
		},
		{
			name: "custom region with base URL is allowed",
			config: &Config{
				AuthToken: "test-token",
				Region:    "custom-region",
				BaseURL:   "https://custom.dash0.example.com",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError {
				if err == nil {
					t.Errorf("Validate() expected error containing %q, got nil", tt.errorMsg)
					return
				}
				if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("Validate() error = %q, want error containing %q", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestCoalesce(t *testing.T) {
	tests := []struct {
		name     string
		values   []string
		expected string
	}{
		{
			name:     "first non-empty",
			values:   []string{"first", "second", "third"},
			expected: "first",
		},
		{
			name:     "skip empty to find non-empty",
			values:   []string{"", "", "third"},
			expected: "third",
		},
		{
			name:     "all empty returns empty",
			values:   []string{"", "", ""},
			expected: "",
		},
		{
			name:     "no values returns empty",
			values:   []string{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := coalesce(tt.values...)
			if result != tt.expected {
				t.Errorf("coalesce(%v) = %q, want %q", tt.values, result, tt.expected)
			}
		})
	}
}

func TestParseBool(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"true", true},
		{"TRUE", true},
		{"True", true},
		{"1", true},
		{"yes", true},
		{"YES", true},
		{"Yes", true},
		{"false", false},
		{"FALSE", false},
		{"0", false},
		{"no", false},
		{"", false},
		{"invalid", false},
		{"  true  ", true},
		{"  false  ", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseBool(tt.input)
			if result != tt.expected {
				t.Errorf("parseBool(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDeriveBaseURL(t *testing.T) {
	tests := []struct {
		region   Region
		expected string
	}{
		{RegionEUWest1, "https://api.eu-west-1.aws.dash0.com"},
		{RegionUSEast1, "https://api.us-east-1.aws.dash0.com"},
		{RegionUSWest2, "https://api.us-west-2.aws.dash0.com"},
		{"unknown", ""},
	}

	for _, tt := range tests {
		t.Run(string(tt.region), func(t *testing.T) {
			cfg := &Config{Region: tt.region}
			result := cfg.deriveBaseURL()
			if result != tt.expected {
				t.Errorf("deriveBaseURL() for region %q = %q, want %q", tt.region, result, tt.expected)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && searchString(s, substr)))
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
