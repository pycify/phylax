package config

import (
	"errors"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Environment string          `json:"environment" mapstructure:"environment"`
	Server      ServerConfig    `json:"server" mapstructure:"server"`
	Storage     StorageConfig   `json:"storage" mapstructure:"storage"`
	Redis       RedisConfig     `json:"redis" mapstructure:"redis"`
	Mail        MailConfig      `json:"mail" mapstructure:"mail"`
	Token       TokenConfig     `json:"token" mapstructure:"token"`
	Challenges  ChallengeConfig `json:"challenges" mapstructure:"challenges"`
}

type ServerConfig struct {
	Addr string `json:"addr" mapstructure:"addr"`
}

type StorageConfig struct {
	Driver      string `json:"driver" mapstructure:"driver"`
	DatabaseURL string `json:"database_url" mapstructure:"database_url"`
}

type RedisConfig struct {
	URL string `json:"url" mapstructure:"url"`
}

type MailConfig struct {
	Driver    string `json:"driver" mapstructure:"driver"`
	FromName  string `json:"from_name" mapstructure:"from_name"`
	FromEmail string `json:"from_email" mapstructure:"from_email"`
	SMTPHost  string `json:"smtp_host" mapstructure:"smtp_host"`
	SMTPPort  int    `json:"smtp_port" mapstructure:"smtp_port"`
	SMTPUser  string `json:"smtp_user" mapstructure:"smtp_user"`
	SMTPPass  string `json:"smtp_pass" mapstructure:"smtp_pass"`
}

type TokenConfig struct {
	Issuer     string `json:"issuer" mapstructure:"issuer"`
	Secret     string `json:"secret" mapstructure:"secret"`
	TTLSeconds int    `json:"ttl_seconds" mapstructure:"ttl_seconds"`
}

type ChallengeConfig struct {
	TTLSeconds int `json:"ttl_seconds" mapstructure:"ttl_seconds"`
}

type AppConfig struct {
	ID       string         `json:"id" mapstructure:"id"`
	Name     string         `json:"name" mapstructure:"name"`
	Android  AndroidConfig  `json:"android" mapstructure:"android"`
	Apple    AppleConfig    `json:"apple" mapstructure:"apple"`
	Firebase FirebaseConfig `json:"firebase" mapstructure:"firebase"`
	Metadata map[string]any `json:"metadata,omitempty" mapstructure:"metadata"`
}

type AndroidConfig struct {
	Enabled                    bool     `json:"enabled" mapstructure:"enabled"`
	Provider                   string   `json:"provider" mapstructure:"provider"`
	PackageName                string   `json:"package_name" mapstructure:"package_name"`
	CloudProjectNumber         string   `json:"cloud_project_number" mapstructure:"cloud_project_number"`
	ServiceAccountJSON         string   `json:"-" mapstructure:"-"`
	ServiceAccountJSONEnv      string   `json:"service_account_json_env" mapstructure:"service_account_json_env"`
	RequirePlayRecognized      bool     `json:"require_play_recognized" mapstructure:"require_play_recognized"`
	RequiredDeviceIntegrity    []string `json:"required_device_integrity" mapstructure:"required_device_integrity"`
	AllowMockVerification      bool     `json:"allow_mock_verification" mapstructure:"allow_mock_verification"`
	ExpectedCertificateSHA256  []string `json:"expected_certificate_sha256" mapstructure:"expected_certificate_sha256"`
	AllowedVersionCodePrefixes []string `json:"allowed_version_code_prefixes" mapstructure:"allowed_version_code_prefixes"`
}

type AppleConfig struct {
	Enabled               bool   `json:"enabled" mapstructure:"enabled"`
	Provider              string `json:"provider" mapstructure:"provider"`
	BundleID              string `json:"bundle_id" mapstructure:"bundle_id"`
	TeamID                string `json:"team_id" mapstructure:"team_id"`
	Environment           string `json:"environment" mapstructure:"environment"`
	AllowMockVerification bool   `json:"allow_mock_verification" mapstructure:"allow_mock_verification"`
}

type FirebaseConfig struct {
	Enabled        bool   `json:"enabled" mapstructure:"enabled"`
	ProjectID      string `json:"project_id" mapstructure:"project_id"`
	CredentialsEnv string `json:"credentials_env" mapstructure:"credentials_env"`
}

func Load() (Config, error) {
	var cfg Config
	cfg.applyDefaults()
	cfg.loadFromEnv()
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c *Config) applyDefaults() {
	if c.Mail.Driver == "" {
		c.Mail.Driver = "log"
	}
	if c.Token.TTLSeconds == 0 {
		c.Token.TTLSeconds = int((15 * time.Minute).Seconds())
	}
	if c.Challenges.TTLSeconds == 0 {
		c.Challenges.TTLSeconds = int((5 * time.Minute).Seconds())
	}
}

func (c *Config) loadFromEnv() {
	if value := os.Getenv("PHYLAX_ENV"); value != "" {
		c.Environment = value
	}
	if value := os.Getenv("PORT"); value != "" {
		c.Server.Addr = ":" + value
	}
	if value := os.Getenv("PHYLAX_ADDR"); value != "" {
		c.Server.Addr = value
	}
	if value := os.Getenv("PHYLAX_STORAGE_DRIVER"); value != "" {
		c.Storage.Driver = value
	}
	if value := os.Getenv("DATABASE_URL"); value != "" {
		c.Storage.DatabaseURL = value
	}
	if value := os.Getenv("REDIS_URL"); value != "" {
		c.Redis.URL = value
	}
	if value := os.Getenv("MAIL_DRIVER"); value != "" {
		c.Mail.Driver = value
	}
	if value := os.Getenv("MAIL_FROM_NAME"); value != "" {
		c.Mail.FromName = value
	}
	if value := os.Getenv("MAIL_FROM_EMAIL"); value != "" {
		c.Mail.FromEmail = value
	}
	if value := os.Getenv("SMTP_HOST"); value != "" {
		c.Mail.SMTPHost = value
	}
	if value := os.Getenv("SMTP_PORT"); value != "" {
		if port, err := strconv.Atoi(value); err == nil {
			c.Mail.SMTPPort = port
		}
	}
	if value := os.Getenv("SMTP_USER"); value != "" {
		c.Mail.SMTPUser = value
	}
	if value := os.Getenv("SMTP_PASS"); value != "" {
		c.Mail.SMTPPass = value
	}
	if value := os.Getenv("PHYLAX_TOKEN_ISSUER"); value != "" {
		c.Token.Issuer = value
	}
	if value := os.Getenv("PHYLAX_TOKEN_SECRET"); value != "" {
		c.Token.Secret = value
	}
	if value := os.Getenv("PHYLAX_TOKEN_TTL_SECONDS"); value != "" {
		if ttl, err := strconv.Atoi(value); err == nil {
			c.Token.TTLSeconds = ttl
		}
	}
	if value := os.Getenv("PHYLAX_CHALLENGE_TTL_SECONDS"); value != "" {
		if ttl, err := strconv.Atoi(value); err == nil {
			c.Challenges.TTLSeconds = ttl
		}
	}
}

func (c Config) Validate() error {
	if c.Environment == "" {
		return errors.New("PHYLAX_ENV is required")
	}
	if c.Environment != "development" && c.Environment != "staging" && c.Environment != "production" {
		return errors.New("environment must be development, staging, or production")
	}
	if c.Storage.Driver != "memory" && c.Storage.Driver != "postgres" {
		return errors.New("storage.driver must be memory or postgres")
	}
	if c.Storage.DatabaseURL == "" {
		return errors.New("DATABASE_URL is required")
	}
	if c.Mail.Driver != "log" && c.Mail.Driver != "smtp" {
		return errors.New("mail.driver must be log or smtp")
	}
	if c.Mail.Driver == "smtp" && (c.Mail.SMTPHost == "" || c.Mail.SMTPPort == 0 || c.Mail.FromEmail == "") {
		return errors.New("smtp mail requires SMTP_HOST, SMTP_PORT, and MAIL_FROM_EMAIL")
	}
	if len(c.Token.Secret) < 32 {
		return errors.New("token.secret must be at least 32 bytes")
	}
	return nil
}
