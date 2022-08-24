package main

import (
	"fmt"
	"oneway-filesync/pkg/database"
	"os"
	"path/filepath"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s <file/dir_path>\n", os.Args[0])
		return
	}
	err := database.ConfigureDatabase()
	if err != nil {
		fmt.Printf("Failed setting up db with err %v\n", err)
		return
	}

	db, err := database.OpenDatabase()
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	path := os.Args[1]
	filepath.Walk(path, func(filepath string, info os.FileInfo, e error) error {
		if !info.IsDir() {
			err := database.QueueFileForSending(db, filepath)
			if err != nil {
				fmt.Printf("%v\n", err)
			} else {
				fmt.Printf("File '%s' queued for sending\n", filepath)
			}
		}
		return nil
	})
}
