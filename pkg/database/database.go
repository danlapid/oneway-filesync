package database

import (
	"fmt"
	"oneway-filesync/pkg/structs"
	"os"
	"path/filepath"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
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

const DBFILE = "gorm.db"

// Opens a connection to the database,
// eventually we can choose to receive the user, password, host, database name
// from the the configuration file, because we expect this database to be run locally
// we leave it as defaults for now.
func OpenDatabase(tableprefix string) (*gorm.DB, error) {
	return gorm.Open(sqlite.Open(DBFILE),
		&gorm.Config{
			NamingStrategy: schema.NamingStrategy{TablePrefix: tableprefix},
			Logger:         gormlogger.Discard,
		})
}

func ConfigureDatabase(db *gorm.DB) error {
	return db.AutoMigrate(&File{})
}

func ClearDatabase(db *gorm.DB) error {
	stmt := &gorm.Statement{DB: db}
	stmt.Parse(&File{})
	tablename := stmt.Schema.Table
	return db.Exec(fmt.Sprintf("DELETE FROM %s", tablename)).Error
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
