package main

import (
	"context"
	"flag"
	"fmt"
	"lazarus/internal/agent"
	"lazarus/internal/agent/tools"
	"lazarus/internal/config"
	"lazarus/internal/knowledge"
	"lazarus/internal/logger"
	"lazarus/internal/provider"
	"lazarus/internal/repository"
	"lazarus/internal/routes"
	docsvc "lazarus/internal/service/artifact_manager"
	"lazarus/internal/service/authorization"
	labsvc "lazarus/internal/service/lab"
	"lazarus/internal/service/push"
	risksvc "lazarus/internal/service/risk"
	"lazarus/internal/service/user"
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
	repo := repository.InitRepo(dbConn)

	appLog.Info("init services")
	srvAuth := authorization.NewService(ctx, appLog, appConf, repo)
	srvUser := user.NewService(ctx, appLog, appConf, repo)

	// MedHelp services
	var medDeps *routes.MedHelpDeps
	db := dbConn.Client()

	if appConf.S3 != nil {
		appLog.Info("init medhelp services")

		// Object storage
		bucketClient, err := bucket.NewClient(ctx, appConf.S3)
		if err != nil {
			appLog.Fatal("unable to connect to object storage", err)
		}

		// LLM provider registry
		providerReg, err := provider.NewRegistry(&appConf.LLM)
		if err != nil {
			appLog.Fatal("unable to init LLM provider registry", err)
		}

		// Knowledge base
		kbRepo := knowledge.NewRepository(db)

		// Lab service
		labService := labsvc.NewService(db, kbRepo)

		// Risk service
		riskService := risksvc.NewService()

		// Document service
		docService := docsvc.NewService(db, bucketClient, providerReg)

		// Patient model store
		patientStore := agent.NewPatientModelStore(db)

		// Session store
		sessionStore := agent.NewSessionStore(db)

		// Tool registry
		toolRegistry := tools.NewRegistry(&tools.Deps{
			DB:          db,
			KBRepo:      kbRepo,
			LabSvc:      labService,
			RiskSvc:     riskService,
			ProviderReg: providerReg,
		})

		// Context assembler
		assembler := agent.NewAssembler(db, patientStore, labService, riskService, kbRepo)

		// Orchestrator
		orchestrator := agent.NewOrchestrator(assembler, providerReg, toolRegistry, sessionStore, patientStore, db)

		medDeps = &routes.MedHelpDeps{
			DB:           db,
			Orchestrator: orchestrator,
			DocSvc:       docService,
			LabSvc:       labService,
		}
	} else {
		appLog.Info("S3 not configured — medhelp services disabled")
	}

	appLog.Info("init http service")
	appHTTPServer := routes.InitAppRouter(
		appLog,
		appConf,
		srvAuth,
		srvUser,
		fmt.Sprintf(":%d", appConf.AppPort),
		true,
		medDeps,
	)
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
