package main

import (
	"context"
	"oneway-filesync/pkg/config"
	"oneway-filesync/pkg/database"
	"oneway-filesync/pkg/utils"
	"oneway-filesync/pkg/watcher"

	"github.com/sirupsen/logrus"
)

func main() {
	utils.InitializeLogging("watcher.log")
	conf, err := config.GetConfig("config.toml")
	if err != nil {
		logrus.Errorf("Failed reading config with err %v", err)
		return
	}

	db, err := database.OpenDatabase("s_")
	if err != nil {
		logrus.Errorf("%v", err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background()) // Create a cancelable context and pass it to all goroutines, allows us to gracefully shut down the program
	watcher.Watcher(ctx, db, conf)

	<-utils.CtrlC()
	cancel() // Gracefully shutdown and stop all goroutines
}
