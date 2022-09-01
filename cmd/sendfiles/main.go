package main

import (
	"fmt"
	"oneway-filesync/pkg/config"
	"oneway-filesync/pkg/database"
	"os"
	"path/filepath"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s <file/dir_path>\n", os.Args[0])
		return
	}

	conf, err := config.GetConfig("config.toml")
	if err != nil {
		fmt.Printf("Failed reading config with err %v", err)
		return
	}

	db, err := database.OpenDatabase("s_")
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}

	path := os.Args[1]
	err = filepath.Walk(path, func(filepath string, info os.FileInfo, e error) error {
		if !info.IsDir() {
			err := database.QueueFileForSending(db, filepath, conf.EncryptedOutput)
			if err != nil {
				fmt.Printf("%v\n", err)
			} else {
				fmt.Printf("File '%s' queued for sending\n", filepath)
			}
		}
		return nil
	})
	if err != nil {
		fmt.Printf("Failed walking dir with err %v\n", err)
		return
	}
}
