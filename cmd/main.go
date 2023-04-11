package main

import (
	"flag"
	"log"

	"github.com/faozimipa/go-cqrs-es/config"
	"github.com/faozimipa/go-cqrs-es/internal/app"
	"github.com/faozimipa/go-cqrs-es/pkg/logger"
)

func main() {
	log.Println("Starting microservice")

	flag.Parse()

	cfg, err := config.InitConfig()
	if err != nil {
		log.Fatal(err)
	}

	appLogger := logger.NewAppLogger(cfg.Logger)
	appLogger.InitLogger()
	appLogger.Named(app.GetMicroserviceName(*cfg))
	appLogger.Infof("CFG: %+v", cfg)
	appLogger.Fatal(app.NewApp(appLogger, *cfg).Run())
}
