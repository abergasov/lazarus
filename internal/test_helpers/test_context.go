package testhelpers

import (
	"context"
	"fmt"
	"lazarus/internal/config"
	"lazarus/internal/logger"
	"lazarus/internal/repository"
	"lazarus/internal/service/artifact_inspector"
	"lazarus/internal/service/artifact_manager"
	"lazarus/internal/service/authorization"
	"lazarus/internal/service/user"
	"lazarus/internal/storage/antivirus"
	"lazarus/internal/storage/bucket"
	"lazarus/internal/storage/database"
	"os"
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
	ServiceAntivirus         *antivirus.Client
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
	srvArtifactManager := artifact_manager.NewService(ctx, appLog, conf, repo, storageClient)
	srvArtifactInspector := artifact_inspector.NewService(ctx, appLog, conf, repo, storageClient, srvAntivirus)
	return &TestContainer{
		Ctx:    ctx,
		Cfg:    conf,
		Logger: appLog,

		Conn:         dbConnect,
		BucketClient: storageClient,

		Repo: repo,

		ServiceAuth:              srvAuth,
		ServiceUser:              srvUser,
		ServiceArtifactManager:   srvArtifactManager,
		ServiceArtifactInspector: srvArtifactInspector,
		ServiceAntivirus:         srvAntivirus,
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
