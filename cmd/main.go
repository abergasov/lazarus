package main

import (
	"context"
	"flag"
	"fmt"
	"lazarus/internal/config"
	"lazarus/internal/logger"
	"lazarus/internal/repository"
	"lazarus/internal/routes"
	"lazarus/internal/service/authorization"
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
	appConf, err := config.InitConf(*confFile)
	if err != nil {
		appLog.Fatal("unable to init config", err, logger.WithString("config", *confFile))
	}
	ctx, cancel := context.WithCancel(context.Background())

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

	appLog.Info("init repositories")
	_ = repository.InitRepo(dbConn)

	appLog.Info("init services")
	service := authorization.NewService(ctx, appConf, appLog)

	appLog.Info("init http service")
	appHTTPServer := routes.InitAppRouter(appLog, appConf, service, fmt.Sprintf(":%d", appConf.AppPort), true)
	defer func() {
		if err = appHTTPServer.Stop(); err != nil {
			appLog.Fatal("unable to stop http service", err)
		}
	}()
	go func() {
		if err = appHTTPServer.Run(); err != nil {
			appLog.Fatal("unable to start http service", err)
		}
	}()

	// register app shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c // This blocks the main thread until an interrupt is received
	cancel()
}
