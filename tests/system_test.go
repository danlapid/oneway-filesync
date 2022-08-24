package main

import (
	"context"
	"crypto/rand"
	"io"
	"log"
	"oneway-filesync/pkg/config"
	"oneway-filesync/pkg/database"
	"oneway-filesync/pkg/receiver"
	"oneway-filesync/pkg/sender"
	"os"
	"testing"
	"time"

	"gorm.io/gorm"
)

func TestSetup(t *testing.T) {
	_, _, teardowntest := setupTest(t, config.Config{
		ReceiverIP:       "127.0.0.1",
		ReceiverPort:     5000,
		BandwidthLimit:   10000,
		ChunkSize:        8192,
		ChunkFecRequired: 5,
		ChunkFecTotal:    8,
		OutDir:           "tests_out",
	})
	defer teardowntest()
}

func TestSmallFile(t *testing.T) {
	senderdb, receiverdb, teardowntest := setupTest(t, config.Config{
		ReceiverIP:       "127.0.0.1",
		ReceiverPort:     5000,
		BandwidthLimit:   10000,
		ChunkSize:        8192,
		ChunkFecRequired: 5,
		ChunkFecTotal:    8,
		OutDir:           "tests_out",
	})
	defer teardowntest()

	testfile := tempFile(t, 500)
	defer os.Remove(testfile)

	database.QueueFileForSending(senderdb, testfile)
	waitForFinishedFile(t, receiverdb, testfile, time.Second*60)
}

func TestLargeFile(t *testing.T) {
	senderdb, receiverdb, teardowntest := setupTest(t, config.Config{
		ReceiverIP:       "127.0.0.1",
		ReceiverPort:     5000,
		BandwidthLimit:   4 * 1024 * 1024,
		ChunkSize:        8192,
		ChunkFecRequired: 5,
		ChunkFecTotal:    8,
		OutDir:           "tests_out",
	})
	defer teardowntest()

	testfile := tempFile(t, 50*1024*1024)
	defer os.Remove(testfile)

	database.QueueFileForSending(senderdb, testfile)
	waitForFinishedFile(t, receiverdb, testfile, time.Second*90)
}

func TestVeryLargeFile(t *testing.T) {
	senderdb, receiverdb, teardowntest := setupTest(t, config.Config{
		ReceiverIP:       "127.0.0.1",
		ReceiverPort:     5000,
		BandwidthLimit:   500 * 1024 * 1024,
		ChunkSize:        8192,
		ChunkFecRequired: 5,
		ChunkFecTotal:    8,
		OutDir:           "tests_out",
	})
	defer teardowntest()

	testfile := tempFile(t, 5*1024*1024*1024)
	defer os.Remove(testfile)

	database.QueueFileForSending(senderdb, testfile)
	waitForFinishedFile(t, receiverdb, testfile, time.Second*90)
}

func waitForFinishedFile(t *testing.T, db *gorm.DB, path string, timeout time.Duration) {
	start := time.Now()
	ticker := time.NewTicker(1 * time.Second)
	for {
		<-ticker.C
		if time.Since(start) > timeout {
			t.Fatalf("File '%s' did not transfer in time", path)
		}
		var file database.File
		err := db.Where("Path = ?", path).First(&file).Error
		if err != nil {
			continue
		}
		if !file.Finished || !file.Success {
			t.Fatalf("File '%s' transferred but not successfully", path)
		} else {
			return
		}
	}
}

func tempFile(t *testing.T, size int) string {
	file, err := os.CreateTemp("", "")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	_, err = io.CopyN(file, rand.Reader, int64(size))
	if err != nil {
		log.Fatal(err)
	}
	return file.Name()
}

func setupTest(t *testing.T, conf config.Config) (*gorm.DB, *gorm.DB, func()) {
	senderdb, err := database.OpenDatabase("t_s_")
	if err != nil {
		t.Fatalf("Failed setting up db with err: %v\n", err)
	}
	if err := database.ConfigureDatabase(senderdb); err != nil {
		t.Fatalf("Failed setting up db with err: %v\n", err)
	}

	receiverdb, err := database.OpenDatabase("t_r_")
	if err != nil {
		t.Fatalf("Failed setting up db with err: %v\n", err)
	}
	if err := database.ConfigureDatabase(receiverdb); err != nil {
		t.Fatalf("Failed setting up db with err: %v\n", err)
	}

	if err := os.MkdirAll(conf.OutDir, os.ModePerm); err != nil {
		t.Fatalf("Failed creating outdir with err: %v\n", err)
	}

	ctx, cancel := context.WithCancel(context.Background()) // Create a cancelable context and pass it to all goroutines, allows us to gracefully shut down the program

	receiver.Receiver(ctx, receiverdb, conf)
	sender.Sender(ctx, senderdb, conf)

	return senderdb, receiverdb, func() {
		cancel()
		os.RemoveAll(conf.OutDir)
		database.ClearDatabase(receiverdb)
		database.ClearDatabase(senderdb)
	}
}
