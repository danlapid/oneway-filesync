package main

import (
	"fmt"
	"oneway-filesync/pkg/database"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s <filepath>\n", os.Args[0])
		return
	}
	err := database.ConfigureDatabase()
	if err != nil {
		fmt.Printf("Failed setting up db with err %v\n", err)
		return
	}

	filepath := os.Args[1]
	err = database.QueueFileForSending(filepath)
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	} else {
		fmt.Printf("File queued for sending\n")
	}
}
