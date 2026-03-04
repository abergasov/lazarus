package config

import (
	"fmt"
	"lazarus/internal/storage/bucket"
	"log"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type AppConfig struct {
	AppPort         int            `yaml:"app_port"`
	EnableTelemetry bool           `yaml:"enable_telemetry"`
	MigratesFolder  string         `yaml:"migrates_folder"`
	ConfigDB        DBConf         `yaml:"conf_db"`
	S3              *bucket.S3Conf `yaml:"s3"`
	AuthConfig      Config         `yaml:"auth"`
}

type Config struct {
	SupabaseURL    string `yaml:"supabase_url"`  // https://<ref>.supabase.co
	SupabaseAnon   string `yaml:"supabase_anon"` // anon key
	CallbackURL    string `yaml:"callback_url"`  // https://app.example.com/auth/callback  (must be in Supabase Redirect URLs allowlist)
	FrontendURL    string `yaml:"frontend_url"`  // https://app.example.com
	CookieDomain   string `yaml:"cookie_domain"` // optional
	CookieSecure   bool   `yaml:"cookie_secure"`
	CookieSameSite string `yaml:"cookie_same_site"` // "lax" | "strict" | "none"
	Mode           string `yaml:"mode"`
}

type DBConf struct {
	Address        string `yaml:"address"`
	Port           string `yaml:"port"`
	User           string `yaml:"user"`
	Pass           string `yaml:"pass"`
	DBName         string `yaml:"db_name"`
	MaxConnections int    `yaml:"max_connections"`
}

func InitConf(confFile string) (*AppConfig, error) {
	file, err := os.Open(filepath.Clean(confFile))
	if err != nil {
		return nil, fmt.Errorf("error open config file: %w", err)
	}
	defer func() {
		if e := file.Close(); e != nil {
			log.Fatal("Error close config file", e)
		}
	}()

	var cfg AppConfig
	if err = yaml.NewDecoder(file).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("error decode config file: %w", err)
	}
	if cfg.S3 == nil {
		return nil, fmt.Errorf("s3 config is required")
	}
	if err = cfg.S3.Validate(); err != nil {
		return nil, fmt.Errorf("error validate s3 section: %w", err)
	}
	return &cfg, nil
}
