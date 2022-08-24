package database

import (
	"oneway-filesync/pkg/structs"
	"os"
	"path/filepath"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type File struct {
	gorm.Model
	Path     string `json:"path"`     // Original file path in source machine
	Hash     []byte `json:"hash"`     // Hash of the file for completeness validation
	Started  bool   `json:"started"`  // Whether or not the file started being sent
	Finished bool   `json:"finished"` // Whether or not the file was sent/recieved successfully
	Success  bool   `json:"success"`  // Whether or not the finish was successfull
}
type ReceivedFile struct {
	File
}

// Opens a connection to the database,
// eventually we can choose to receive the user, password, host, database name
// from the the configuration file, because we expect this database to be run locally
// we leave it as defaults for now.
func OpenDatabase() (*gorm.DB, error) {
	dbURL := "postgres://postgres:postgres@localhost:5432/postgres"
	return gorm.Open(postgres.Open(dbURL), &gorm.Config{})
}

func ConfigureDatabase() error {
	db, err := OpenDatabase()
	if err != nil {
		return err
	}
	err = db.AutoMigrate(&File{})
	return err
}

// Receives a file path, hashes it and pushes it into the database
// This should be run from an external program on the source machine
// The sender reads files from this database and sends them.
func QueueFileForSending(db *gorm.DB, path string) error {
	path, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	hash, err := structs.HashFile(f)
	if err != nil {
		return err
	}

	file := File{
		Path:     path,
		Hash:     hash[:],
		Finished: false,
		Success:  false,
	}

	result := db.Create(&file)
	return result.Error
}
