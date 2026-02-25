// Package config provides configuration management for the Dash0 MCP server.
package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// Region represents a Dash0 deployment region.
type Region string

const (
	// RegionEUWest1 is the EU West 1 (Ireland) region.
	RegionEUWest1 Region = "eu-west-1"
	// RegionUSEast1 is the US East 1 (Virginia) region.
	RegionUSEast1 Region = "us-east-1"
	// RegionUSWest2 is the US West 2 (Oregon) region.
	RegionUSWest2 Region = "us-west-2"
)

// Config holds the Dash0 MCP server configuration.
type Config struct {
	// BaseURL is the Dash0 API base URL.
	BaseURL string
	// AuthToken is the Bearer token for authentication.
	AuthToken string
	// Region is the Dash0 deployment region.
	Region Region
	// Dataset is the Dash0 dataset to use for all API calls.
	Dataset string
	// Debug enables debug logging.
	Debug bool
}

// Load reads configuration from environment variables.
// Environment variables:
//   - DASH0_AUTH_TOKEN (required): Bearer token for API authentication
//   - DASH0_REGION (optional): Region (eu-west-1, us-east-1, us-west-2), defaults to eu-west-1
//   - DASH0_BASE_URL (optional): Override the base URL (for custom deployments)
//   - DASH0_DATASET (optional): Dataset to use for all API calls
//   - DASH0_DEBUG (optional): Enable debug logging
func Load() (*Config, error) {
	regionEnv := coalesce(os.Getenv("DASH0_REGION"), string(RegionEUWest1))
	baseURL := os.Getenv("DASH0_BASE_URL")

	// Handle case where full URL is passed as DASH0_REGION
	if strings.HasPrefix(regionEnv, "api.") || strings.HasPrefix(regionEnv, "https://") {
		// Extract region from URL-like value or use as base URL
		if strings.HasPrefix(regionEnv, "https://") {
			baseURL = regionEnv
		} else {
			baseURL = "https://" + regionEnv
		}
		// Try to extract region from URL
		if strings.Contains(regionEnv, "eu-west-1") {
			regionEnv = "eu-west-1"
		} else if strings.Contains(regionEnv, "us-east-1") {
			regionEnv = "us-east-1"
		} else if strings.Contains(regionEnv, "us-west-2") {
			regionEnv = "us-west-2"
		}
	}

	cfg := &Config{
		AuthToken: coalesce(os.Getenv("DASH0_AUTH_TOKEN"), os.Getenv("DASH0_TOKEN")),
		Region:    Region(regionEnv),
		BaseURL:   baseURL,
		Dataset:   os.Getenv("DASH0_DATASET"),
		Debug:     parseBool(os.Getenv("DASH0_DEBUG")),
	}

	// Derive base URL from region if not explicitly set
	if cfg.BaseURL == "" {
		cfg.BaseURL = cfg.deriveBaseURL()
	}

	return cfg, nil
}

// Validate checks that all required configuration is present and valid.
func (c *Config) Validate() error {
	if c.AuthToken == "" {
		return errors.New("DASH0_AUTH_TOKEN is required")
	}

	if c.BaseURL == "" {
		return errors.New("unable to determine base URL: set DASH0_REGION or DASH0_BASE_URL")
	}

	if !strings.HasPrefix(c.BaseURL, "https://") {
		return fmt.Errorf("base URL must use HTTPS: %s", c.BaseURL)
	}

	switch c.Region {
	case RegionEUWest1, RegionUSEast1, RegionUSWest2:
		// Valid regions
	default:
		if c.BaseURL == "" {
			return fmt.Errorf("invalid region %q: must be eu-west-1, us-east-1, or us-west-2", c.Region)
		}
		// Allow custom regions if base URL is explicitly set
	}

	return nil
}

// deriveBaseURL returns the API base URL for the configured region.
func (c *Config) deriveBaseURL() string {
	switch c.Region {
	case RegionEUWest1:
		return "https://api.eu-west-1.aws.dash0.com"
	case RegionUSEast1:
		return "https://api.us-east-1.aws.dash0.com"
	case RegionUSWest2:
		return "https://api.us-west-2.aws.dash0.com"
	default:
		return ""
	}
}

// coalesce returns the first non-empty string.
func coalesce(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// parseBool parses a boolean from a string, returning false for invalid values.
func parseBool(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "true" || s == "1" || s == "yes"
}
