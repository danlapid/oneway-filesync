package main

import (
	"context"
	"oneway-filesync/pkg/database"
	"oneway-filesync/pkg/utils"
	"oneway-filesync/pkg/watcher"
	"os"
	"os/signal"
	"syscall"

	"github.com/rjeczalik/notify"
	"github.com/sirupsen/logrus"
)

func main() {
	utils.InitializeLogging("watcher.log")
	if len(os.Args) < 2 {
		logrus.Errorf("Usage: %s <dir_path>", os.Args[0])
		return
	}
	path := os.Args[1]

	db, err := database.OpenDatabase("s_")
	if err != nil {
		logrus.Errorf("%v", err)
		return
	}

	if err = database.ConfigureDatabase(db); err != nil {
		logrus.Errorf("Failed setting up db with err %v", err)
		return
	}

	events := make(chan notify.EventInfo, 20)
	ctx, cancel := context.WithCancel(context.Background()) // Create a cancelable context and pass it to all goroutines, allows us to gracefully shut down the program
	watcher.CreateWatcher(ctx, db, path, events)

	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)
	<-done
	cancel() // Gracefully shutdown and stop all goroutines
}
