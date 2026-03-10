package testhelpers

import (
	"context"
	"fmt"
	"lazarus/internal/config"
	"lazarus/internal/entities"
	"lazarus/internal/logger"
	"lazarus/internal/repository"
	"lazarus/internal/service/artifact_inspector"
	"lazarus/internal/service/artifact_manager"
	"lazarus/internal/service/artifact_parser"
	"lazarus/internal/service/authorization"
	"lazarus/internal/service/provider"
	"lazarus/internal/service/user"
	"lazarus/internal/storage/antivirus"
	"lazarus/internal/storage/bucket"
	"lazarus/internal/storage/database"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type TestContainer struct {
	Ctx    context.Context
	Cfg    *config.AppConfig
	Logger logger.AppLogger

	Conn         database.DBConnector
	BucketClient *bucket.Client

	Repo *repository.Repo

	ServiceAuth              *authorization.Service
	ServiceUser              *user.Service
	ServiceArtifactManager   *artifact_manager.Service
	ServiceArtifactInspector *artifact_inspector.Service
	ServiceArtifactParser    *artifact_parser.Service
	ServiceAntivirus         *antivirus.Client
	ServiceRegistry          *provider.Registry
}

func GetClean(t *testing.T) *TestContainer {
	return GetCleanWithConfig(t, GetTestConfig(t))
}

func GetCleanWithConfig(t *testing.T, conf *config.AppConfig) *TestContainer {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	prepareTestDB(ctx, t, &conf.ConfigDB)

	dbConnect, err := database.InitDBConnect(ctx, &conf.ConfigDB, guessMigrationDir(t))
	require.NoError(t, err)
	cleanupDB(t, dbConnect)
	storageClient, err := bucket.NewClient(ctx, conf.S3)
	require.NoError(t, err)
	t.Cleanup(func() {
		cancel()
		require.NoError(t, dbConnect.Client().Close())
	})

	appLog := logger.NewAppSLogger()
	// repo init
	repo := repository.InitRepo(dbConnect)

	// service init
	srvAntivirus := antivirus.NewClient(conf.ClamavURL, 1*time.Minute)
	srvAuth := authorization.NewService(ctx, appLog, conf, repo)
	srvUser := user.NewService(ctx, appLog, conf, repo)
	srvArtifactManager, err := artifact_manager.NewService(ctx, appLog, conf, repo, storageClient)
	require.NoError(t, err)
	srvArtifactInspector := artifact_inspector.NewService(ctx, appLog, conf, repo, storageClient, srvAntivirus)
	srvProviderRegistry, err := provider.NewRegistry(ctx, appLog, conf, repo)
	require.NoError(t, err)
	srvArtifactParser := artifact_parser.NewService(ctx, appLog, conf, repo, storageClient, srvProviderRegistry)
	return &TestContainer{
		Ctx:          ctx,
		Cfg:          conf,
		Logger:       appLog,
		Conn:         dbConnect,
		BucketClient: storageClient,

		Repo: repo,

		ServiceAuth:              srvAuth,
		ServiceUser:              srvUser,
		ServiceArtifactManager:   srvArtifactManager,
		ServiceArtifactInspector: srvArtifactInspector,
		ServiceAntivirus:         srvAntivirus,
		ServiceArtifactParser:    srvArtifactParser,
		ServiceRegistry:          srvProviderRegistry,
	}
}

func prepareTestDB(ctx context.Context, t *testing.T, cnf *config.DBConf) {
	dbConnect, err := database.InitDBConnect(ctx, &config.DBConf{
		Address:        cnf.Address,
		Port:           cnf.Port,
		User:           cnf.User,
		Pass:           cnf.Pass,
		DBName:         "postgres",
		MaxConnections: cnf.MaxConnections,
	}, "")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, dbConnect.Client().Close())
	}()
	if _, err = dbConnect.Client().Exec(fmt.Sprintf("CREATE DATABASE %s", cnf.DBName)); !isDatabaseExists(err) {
		require.NoError(t, err)
	}
}

func GetActualConfig(t *testing.T) *config.AppConfig {
	path, err := os.Getwd()
	require.NoError(t, err)
	confPath := filepath.Join(strings.Split(path, "internal")[0], "configs/app_conf.yml") //nolint:gocritic // it ok
	cfg, err := config.InitConf(confPath)
	require.NoError(t, err)
	return cfg
}

func GetTestConfig(t *testing.T) *config.AppConfig {
	return &config.AppConfig{
		AppPort: 0,
		ConfigDB: config.DBConf{
			Address:        "localhost",
			Port:           "5559",
			User:           "aHAjeK",
			Pass:           "AOifjwelmc8dw",
			DBName:         "lazarus_test",
			MaxConnections: 10,
		},
		ClamavURL:     "127.0.0.1:3310",
		RawUploadsDir: t.TempDir(),
		S3: &bucket.S3Conf{
			Region:             "us-east-1",
			Endpoint:           "http://127.0.0.1:9000",
			Bucket:             "lazarus",
			AccessKeyID:        "minioadmin",
			SecretAccessKey:    "minioadmin",
			Prefix:             "test/",
			UsePathStyle:       true,
			MaxUploadSizeBytes: 1024,
		},
		LLM: config.LLMConfig{
			Providers: []*entities.AIProvider{
				{
					Type:         entities.AgentProviderAnthropic,
					APIKey:       os.Getenv("LAZARUS_ANTHROPIC_API_KEY"),
					DefaultModel: "claude-sonnet-4-6",
				},
				{
					Type:         entities.AgentProviderOpenAI,
					APIKey:       os.Getenv("LAZARUS_OPENAI_API_KEY"),
					DefaultModel: "gpt-4.1",
				},
			},
			Roles: entities.RoleConfig{
				PrepVisit: &entities.AgentRoleConfig{
					ProviderID: entities.AgentProviderOpenAI,
					Model:      "gpt-4.1-mini",
				},
				DuringVisit: &entities.AgentRoleConfig{
					ProviderID: entities.AgentProviderOpenAI,
					Model:      "gpt-4.1-mini",
				},
				AfterVisit: &entities.AgentRoleConfig{
					ProviderID: entities.AgentProviderOpenAI,
					Model:      "gpt-4.1-mini",
				},
				Vision: &entities.AgentRoleConfig{
					ProviderID: entities.AgentProviderOpenAI,
					Model:      "gpt-4.1-mini",
				},
				Embed: &entities.AgentRoleConfig{
					ProviderID: entities.AgentProviderOpenAI,
					Model:      "text-embedding-3-small",
				},
			},
		},
	}
}

func isDatabaseExists(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "42P04") || strings.Contains(err.Error(), "23505")
}

func guessMigrationDir(t *testing.T) string {
	dir, err := os.Getwd()
	require.NoError(t, err)
	res := strings.Split(dir, "/internal")
	return res[0] + "/migrations"
}

func cleanupDB(t *testing.T, connector database.DBConnector) {
	for _, table := range repository.AllTables {
		_, err := connector.Client().Exec(fmt.Sprintf("TRUNCATE %s CASCADE", table))
		require.NoError(t, err)
	}
}
