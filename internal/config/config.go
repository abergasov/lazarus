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
	AppDomain          string         `yaml:"app_domain"`
	FrontendURL        string         `yaml:"frontend_url"`
	SSLEnable          bool           `yaml:"ssl_enable"`
	JWTKey             string         `yaml:"jwt_key"` //nolint:gosec // it ok
	JWTLive            int64          `yaml:"jwt_live"`
	GoogleAppSecret    string         `yaml:"google_app_secret"`
	GoogleAppID        string         `yaml:"google_app_id"`
	AppPort            int            `yaml:"app_port"`
	RawUploadsDir      string         `yaml:"raw_uploads_dir"`       // dir for storing raw uploaded files, which will be deleted after processing
	MaxUploadSizeBytes int64          `yaml:"max_upload_size_bytes"` // in bytes how max file size can be uploaded, e.g. 10*1024*1024 for 10MB
	EnableTelemetry    bool           `yaml:"enable_telemetry"`
	MigratesFolder     string         `yaml:"migrates_folder"`
	ConfigDB           DBConf         `yaml:"conf_db"`
	S3                 *bucket.S3Conf `yaml:"s3"`
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
