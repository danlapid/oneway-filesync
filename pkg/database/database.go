package database

import (
	"io"
	"oneway-filesync/pkg/structs"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type File struct {
	gorm.Model
	Path     string `json:"path"`     // Original file path in source machine
	Size     int64  `json:"size"`     // Size of the file, required for unpadding
	Hash     []byte `json:"hash"`     // Hash of the file for completeness validation
	Started  bool   `json:"started"`  // Whether or not the file started being sent
	Finished bool   `json:"finished"` // Whether or not the file was sent/recieved successfully
	Success  bool   `json:"success"`  // Whether or not the finish was successfull
}

// Opens a connection to the database,
// eventually we can choose to receive the user, password, host, database name
// from the the configuration file, because we expect this database to be run locally
// we leave it as defaults for now.
func OpenDatabase() (*gorm.DB, error) {
	dbURL := "postgres://postgres:postgres@localhost:5432/postgres"
	db, err := gorm.Open(postgres.Open(dbURL), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	err = db.AutoMigrate(&File{})
	return db, err
}

// Receives a file path, hashes it and pushes it into the database
// This should be run from an external program on the source machine
// The sender reads files from this database and sends them.
func QueueFileForSending(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return err
	}

	h := structs.HashNew()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	file := File{
		Path:     path,
		Size:     fi.Size(),
		Hash:     h.Sum(nil),
		Finished: false,
		Success:  false,
	}
	db, err := OpenDatabase()
	if err != nil {
		return err
	}
	result := db.Create(&file)
	return result.Error
}

func UpdateFileInDatabase(file File) error {
	db, err := OpenDatabase()
	if err != nil {
		return err
	}
	return db.Save(&file).Error
}
