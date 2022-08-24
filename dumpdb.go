package main

import (
	"fmt"
	"oneway-filesync/pkg/database"
)

func main() {
	var files []database.File
	db, _ := database.OpenDatabase("s_")
	db.Limit(1000).Find(&files)
	for _, file := range files {
		fmt.Printf("%s %t %t %t\n", file.Path, file.Started, file.Finished, file.Success)
	}
}
