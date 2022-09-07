package database

import (
	"fmt"
	"oneway-filesync/pkg/structs"
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

type File struct {
	gorm.Model
	Path      string `json:"path"`      // Original file path in source machine
	Hash      []byte `json:"hash"`      // Hash of the file for completeness validation
	Encrypted bool   `json:"encrypted"` // Whether or not the file is packed as zip
	Started   bool   `json:"started"`   // Whether or not the file started being sent
	Finished  bool   `json:"finished"`  // Whether or not the file was sent/recieved successfully
	Success   bool   `json:"success"`   // Whether or not the finish was successfull
}
type ReceivedFile struct {
	File
}

const DBFILE = "gorm.db?cache=shared&mode=rwc&_journal_mode=WAL&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"

func configureDatabase(db *gorm.DB) error {
	return db.AutoMigrate(&File{})
}

// Opens a connection to the database,
// eventually we can choose to receive the user, password, host, database name
// from the the configuration file, because we expect this database to be run locally
// we leave it as defaults for now.
func OpenDatabase(tableprefix string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(DBFILE),
		&gorm.Config{
			NamingStrategy: schema.NamingStrategy{TablePrefix: tableprefix},
			Logger:         gormlogger.Discard,
		})
	if err != nil {
		return nil, err
	}
	if err = configureDatabase(db); err != nil {
		return nil, err
	}
	return db, nil
}

func ClearDatabase(db *gorm.DB) error {
	stmt := &gorm.Statement{DB: db}
	err := stmt.Parse(&File{})
	if err != nil {
		return err
	}
	tablename := stmt.Schema.Table
	return db.Exec(fmt.Sprintf("DELETE FROM %s", tablename)).Error
}

// Receives a file path, hashes it and pushes it into the database
// This should be run from an external program on the source machine
// The sender reads files from this database and sends them.
func QueueFileForSending(db *gorm.DB, path string, encrypted bool) error {
	path, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	hash, err := structs.HashFile(f, encrypted)
	if err != nil {
		return err
	}

	file := File{
		Path:      path,
		Hash:      hash[:],
		Encrypted: encrypted,
		Started:   false,
		Finished:  false,
		Success:   false,
	}

	return db.Create(&file).Error
}
