package main

import (
	"context"
	"oneway-filesync/pkg/config"
	"oneway-filesync/pkg/database"
	"oneway-filesync/pkg/sender"
	"oneway-filesync/pkg/utils"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
)

func main() {
	conf, err := config.GetConfig("config.toml")
	if err != nil {
		logrus.Errorf("Failed reading config with err %v\n", err)
		return
	}

	db, err := database.OpenDatabase("s_")
	if err != nil {
		logrus.Errorf("Failed connecting to db with err %v\n", err)
		return
	}

	if err = database.ConfigureDatabase(db); err != nil {
		logrus.Errorf("Failed setting up db with err %v\n", err)
		return
	}

	utils.InitializeLogging("sender.log")

	ctx, cancel := context.WithCancel(context.Background()) // Create a cancelable context and pass it to all goroutines, allows us to gracefully shut down the program
	sender.Sender(ctx, db, conf)

	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)
	<-done
	cancel() // Gracefully shutdown and stop all goroutines
}
