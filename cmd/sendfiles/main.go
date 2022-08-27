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

	db, err := database.OpenDatabase("s_")
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}

	if err = database.ConfigureDatabase(db); err != nil {
		fmt.Printf("Failed setting up db with err %v\n", err)
		return
	}

	path := os.Args[1]
	err = filepath.Walk(path, func(filepath string, info os.FileInfo, e error) error {
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
	if err != nil {
		fmt.Printf("Failed walking dir with err %v\n", err)
		return
	}
}
