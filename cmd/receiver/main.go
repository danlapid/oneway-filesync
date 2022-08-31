package main

import (
	"context"
	"oneway-filesync/pkg/config"
	"oneway-filesync/pkg/database"
	"oneway-filesync/pkg/receiver"
	"oneway-filesync/pkg/utils"

	"github.com/sirupsen/logrus"
)

func main() {
	utils.InitializeLogging("receiver.log")
	conf, err := config.GetConfig("config.toml")
	if err != nil {
		logrus.Errorf("Failed reading config with err %v", err)
		return

	}

	db, err := database.OpenDatabase(database.DBFILE, "r_")
	if err != nil {
		logrus.Errorf("Failed connecting to db with err %v", err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background()) // Create a cancelable context and pass it to all goroutines, allows us to gracefully shut down the program
	receiver.Receiver(ctx, db, conf)

	<-utils.CtrlC()
	cancel() // Gracefully shutdown and stop all goroutines
}
