package main

import (
	"context"
	"flag"
	"lazarus/internal/config"
	"lazarus/internal/logger"
	"lazarus/internal/repository"
	"lazarus/internal/service/artifact_inspector"
	"lazarus/internal/storage/bucket"
	"lazarus/internal/storage/database"
	"os"
	"os/signal"
	"syscall"
)

var (
	confFile = flag.String("config", "configs/app_conf.yml", "Configs file path")
)

func main() {
	flag.Parse()
	appLog := logger.NewAppSLogger()
	appLog.Info("app starting", logger.WithString("conf", *confFile))
	ctx, cancel := context.WithCancel(context.Background())
	appConf, err := config.InitConf(*confFile)
	if err != nil {
		appLog.Fatal("failed parse config", err)
	}

	appLog.Info("create storage connections")
	dbConn, err := database.GetDBConnect(ctx, appLog, &appConf.ConfigDB, appConf.MigratesFolder)
	if err != nil {
		appLog.Fatal("unable to connect to db", err, logger.WithString("host", appConf.ConfigDB.Address))
	}
	defer func() {
		if err = dbConn.Close(); err != nil {
			appLog.Fatal("unable to close db connection", err)
		}
	}()
	storageClient, err := bucket.NewClient(ctx, appConf.S3)
	if err != nil {
		appLog.Fatal("unable to create storage client", err)
	}

	appLog.Info("init services")
	srvInspector := artifact_inspector.NewService(ctx, appLog, appConf, repository.InitRepo(dbConn), storageClient)
	go srvInspector.Run()

	// register app shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c // This blocks the main thread until an interrupt is received
	srvInspector.Stop()
	cancel()
}
