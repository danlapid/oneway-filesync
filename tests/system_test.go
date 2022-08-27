package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"oneway-filesync/pkg/config"
	"oneway-filesync/pkg/database"
	"oneway-filesync/pkg/receiver"
	"oneway-filesync/pkg/sender"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"
)

func getDiff(t *testing.T, path1 string, path2 string) int {
	diff := 0
	buf1 := make([]byte, 64*1024)
	buf2 := make([]byte, 64*1024)
	file1, err := os.Open(path1)
	if err != nil {
		t.Fatal(err)
	}
	defer file1.Close()
	file2, err := os.Open(path2)
	if err != nil {
		t.Fatal(err)
	}
	defer file2.Close()
	for {
		nr1, err := file1.Read(buf1)
		if err != nil {
			if err != io.EOF {
				t.Fatal(err)
			}
			break
		}
		nr2, err := file2.Read(buf2)
		if err != nil {
			if err != io.EOF {
				t.Fatal(err)
			}
			break
		}
		if nr1 != nr2 {
			t.Fatal("Different file sizes compared")
		}
		for i, b := range buf1 {
			if b != buf2[i] {
				diff += 1
			}
		}
	}
	return diff
}

func waitForFinishedFile(t *testing.T, db *gorm.DB, path string, timeout time.Duration, outdir string) {
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
			tmpfilepath := filepath.Join(outdir, "tempfiles", fmt.Sprintf("%s___%x.tmp", strings.ReplaceAll(file.Path, "/", "_"), file.Hash))
			diff := getDiff(t, path, tmpfilepath)
			t.Fatalf("File '%s' transferred but not successfully %d different bytes", path, diff)
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
		if err := os.RemoveAll(conf.OutDir); err != nil {
			t.Log(err)
		}
		if err := database.ClearDatabase(receiverdb); err != nil {
			t.Log(err)
		}
		if err := database.ClearDatabase(senderdb); err != nil {
			t.Log(err)
		}
		if err := os.Remove(database.DBFILE); err != nil {
			t.Log(err)
		}
	}
}

func TestSetup(t *testing.T) {
	_, _, teardowntest := setupTest(t, config.Config{
		ReceiverIP:       "127.0.0.1",
		ReceiverPort:     5000,
		BandwidthLimit:   10000,
		ChunkSize:        8192,
		ChunkFecRequired: 5,
		ChunkFecTotal:    10,
		OutDir:           "tests_out",
	})
	defer teardowntest()
}

func TestSmallFile(t *testing.T) {
	conf := config.Config{
		ReceiverIP:       "127.0.0.1",
		ReceiverPort:     5000,
		BandwidthLimit:   10000,
		ChunkSize:        8192,
		ChunkFecRequired: 5,
		ChunkFecTotal:    10,
		OutDir:           "tests_out",
	}
	senderdb, receiverdb, teardowntest := setupTest(t, conf)
	defer teardowntest()

	testfile := tempFile(t, 500)
	defer os.Remove(testfile)

	err := database.QueueFileForSending(senderdb, testfile)
	if err != nil {
		t.Fatal(err)
	}
	waitForFinishedFile(t, receiverdb, testfile, time.Minute, conf.OutDir)
}

func TestLargeFile(t *testing.T) {
	conf := config.Config{
		ReceiverIP:       "127.0.0.1",
		ReceiverPort:     5000,
		BandwidthLimit:   4 * 1024 * 1024,
		ChunkSize:        8192,
		ChunkFecRequired: 5,
		ChunkFecTotal:    10,
		OutDir:           "tests_out",
	}
	senderdb, receiverdb, teardowntest := setupTest(t, conf)
	defer teardowntest()

	testfile := tempFile(t, 50*1024*1024)
	defer os.Remove(testfile)

	err := database.QueueFileForSending(senderdb, testfile)
	if err != nil {
		t.Fatal(err)
	}
	waitForFinishedFile(t, receiverdb, testfile, time.Minute*2, conf.OutDir)
}

func TestVeryLargeFile(t *testing.T) {
	conf := config.Config{
		ReceiverIP:       "127.0.0.1",
		ReceiverPort:     5000,
		BandwidthLimit:   10 * 1024 * 1024,
		ChunkSize:        8192,
		ChunkFecRequired: 5,
		ChunkFecTotal:    10,
		OutDir:           "tests_out",
	}
	senderdb, receiverdb, teardowntest := setupTest(t, conf)
	defer teardowntest()

	testfile := tempFile(t, 1*1024*1024*1024)
	defer os.Remove(testfile)

	err := database.QueueFileForSending(senderdb, testfile)
	if err != nil {
		t.Fatal(err)
	}
	waitForFinishedFile(t, receiverdb, testfile, time.Minute*20, conf.OutDir)
}
