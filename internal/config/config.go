package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"
)

type Config struct {
	Server     ServerConfig    `json:"server"`
	Token      TokenConfig     `json:"token"`
	Challenges ChallengeConfig `json:"challenges"`
	Apps       []AppConfig     `json:"apps"`
}

type ServerConfig struct {
	Addr string `json:"addr"`
}

type TokenConfig struct {
	Issuer     string `json:"issuer"`
	Secret     string `json:"secret"`
	TTLSeconds int    `json:"ttl_seconds"`
}

type ChallengeConfig struct {
	TTLSeconds int `json:"ttl_seconds"`
}

type AppConfig struct {
	ID       string          `json:"id"`
	Name     string          `json:"name"`
	Android  AndroidConfig   `json:"android"`
	Apple    AppleConfig     `json:"apple"`
	Firebase FirebaseConfig  `json:"firebase"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

type AndroidConfig struct {
	Enabled                    bool     `json:"enabled"`
	Provider                   string   `json:"provider"`
	PackageName                string   `json:"package_name"`
	ServiceAccountJSONEnv      string   `json:"service_account_json_env"`
	RequirePlayRecognized      bool     `json:"require_play_recognized"`
	RequiredDeviceIntegrity    []string `json:"required_device_integrity"`
	AllowMockVerification      bool     `json:"allow_mock_verification"`
	ExpectedCertificateSHA256  []string `json:"expected_certificate_sha256"`
	AllowedVersionCodePrefixes []string `json:"allowed_version_code_prefixes"`
}

type AppleConfig struct {
	Enabled               bool   `json:"enabled"`
	Provider              string `json:"provider"`
	BundleID              string `json:"bundle_id"`
	TeamID                string `json:"team_id"`
	Environment           string `json:"environment"`
	AllowMockVerification bool   `json:"allow_mock_verification"`
}

type FirebaseConfig struct {
	Enabled        bool   `json:"enabled"`
	ProjectID      string `json:"project_id"`
	CredentialsEnv string `json:"credentials_env"`
}

func Load(path string) (Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return Config{}, err
	}
	defer file.Close()

	var cfg Config
	if err := json.NewDecoder(file).Decode(&cfg); err != nil {
		return Config{}, err
	}
	cfg.applyDefaults()
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c *Config) applyDefaults() {
	if c.Server.Addr == "" {
		c.Server.Addr = ":8080"
	}
	if c.Token.Issuer == "" {
		c.Token.Issuer = "phylax"
	}
	if c.Token.TTLSeconds == 0 {
		c.Token.TTLSeconds = int((15 * time.Minute).Seconds())
	}
	if c.Challenges.TTLSeconds == 0 {
		c.Challenges.TTLSeconds = int((5 * time.Minute).Seconds())
	}
}

func (c Config) Validate() error {
	if len(c.Token.Secret) < 32 {
		return errors.New("token.secret must be at least 32 bytes")
	}
	if len(c.Apps) == 0 {
		return errors.New("at least one app must be configured")
	}

	seen := map[string]bool{}
	for _, app := range c.Apps {
		if app.ID == "" {
			return errors.New("apps[].id is required")
		}
		if seen[app.ID] {
			return fmt.Errorf("duplicate app id %q", app.ID)
		}
		seen[app.ID] = true

		if app.Android.Enabled && app.Android.PackageName == "" {
			return fmt.Errorf("app %q android.package_name is required", app.ID)
		}
		if app.Apple.Enabled && (app.Apple.BundleID == "" || app.Apple.TeamID == "") {
			return fmt.Errorf("app %q apple.bundle_id and apple.team_id are required", app.ID)
		}
		if app.Firebase.Enabled && app.Firebase.ProjectID == "" {
			return fmt.Errorf("app %q firebase.project_id is required", app.ID)
		}
	}
	return nil
}
