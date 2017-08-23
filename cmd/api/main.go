package main

import (
	"flag"
	"fmt"
	"github.com/facebookgo/grace/gracehttp"
	"github.com/moira-alert/moira-alert/api/handler"
	"github.com/moira-alert/moira-alert/cmd"
	"github.com/moira-alert/moira-alert/database/redis"
	"github.com/moira-alert/moira-alert/logging/go-logging"
	"github.com/moira-alert/moira-alert/metrics/graphite/go-metrics"
	"net/http"
	"os"
)

var (
	configFileName         = flag.String("config", "/etc/moira/config.yml", "Path to configuration file")
	printVersion           = flag.Bool("version", false, "Print version and exit")
	printDefaultConfigFlag = flag.Bool("default-config", false, "Print default config and exit")
	verbosityLog           = flag.Bool("-v", false, "Verbosity log")

	MoiraVersion = "unknown"
	GitCommit    = "unknown"
	Version      = "unknown"
)

func main() {
	flag.Parse()
	if *printVersion {
		fmt.Println("Moira Cache")
		fmt.Println("Version:", MoiraVersion)
		fmt.Println("Git Commit:", GitCommit)
		fmt.Println("Go Version:", Version)
		os.Exit(0)
	}

	config := getDefault()
	if *printDefaultConfigFlag {
		cmd.PrintConfig(config)
		os.Exit(0)
	}

	err := cmd.ReadConfig(*configFileName, &config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can not read settings: %s\n", err.Error())
		os.Exit(1)
	}

	loggerSettings := config.Logger.GetSettings()

	logger, err := logging.ConfigureLog(&loggerSettings, "api")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can not configure log: %s\n", err.Error())
		os.Exit(1)
	}

	databaseSettings := config.Redis.GetSettings()
	databaseMetrics := metrics.ConfigureDatabaseMetrics()
	database := redis.NewDatabase(logger, databaseSettings, databaseMetrics)

	httpHandler := handler.NewHandler(database, logger)

	logger.Infof("Start listening by port: [%s]", config.Api.Port)
	server := &http.Server{
		Addr:    ":" + config.Api.Port,
		Handler: httpHandler,
	}
	if err = gracehttp.Serve(server); err != nil {
		logger.Fatalf("gracehttp failed", err.Error())
	}
	//server.ListenAndServe() //for windows developers =)
	logger.Infof("Stop Moira api")
}
