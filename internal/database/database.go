package database

import (
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/neicnordic/sda-download/internal/config"
	log "github.com/sirupsen/logrus"

	// enables postgres driver
	_ "github.com/lib/pq"
)

// DB is exported for other packages
var DB *SQLdb

// Database defines methods to be implemented by SQLdb
type Database interface {
	GetHeader(fileID string) ([]byte, error)
	GetFile(fileID string) ([]*FileInfo, error)
	Close()
}

// SQLdb struct that acts as a receiver for the DB update methods
type SQLdb struct {
	DB       *sql.DB
	ConnInfo string
}

// FileInfo is returned by the metadata endpoint
type FileInfo struct {
	FileID                    string `json:"fileId"`
	DatasetID                 string `json:"datasetId"`
	DisplayFileName           string `json:"displayFileName"`
	FileName                  string `json:"fileName"`
	FileSize                  int64  `json:"fileSize"`
	DecryptedFileSize         int64  `json:"decryptedFileSize"`
	DecryptedFileChecksum     string `json:"decryptedFileChecksum"`
	DecryptedFileChecksumType string `json:"decryptedFileChecksumType"`
	Status                    string `json:"fileStatus"`
}

// dbRetryTimes is the number of times to retry the same function if it fails
var dbRetryTimes = 3

// dbReconnectTimeout is how long to try to re-establish a connection to the database
var dbReconnectTimeout = 5 * time.Minute

// dbReconnectSleep is how long to wait between attempts to connect to the database
var dbReconnectSleep = 5 * time.Second

// sqlOpen is an internal variable to ease testing
var sqlOpen = sql.Open

// logFatalf is an internal variable to ease testing
var logFatalf = log.Fatalf

// NewDB creates a new DB connection
func NewDB(config config.DatabaseConfig) (*SQLdb, error) {
	connInfo := buildConnInfo(config)

	log.Debugf("Connecting to DB %s:%d on database: %s with user: %s", config.Host, config.Port, config.Database, config.User)
	db, err := sqlOpen("postgres", connInfo)
	if err != nil {
		return nil, err
	}

	if err = db.Ping(); err != nil {
		return nil, err
	}

	return &SQLdb{DB: db, ConnInfo: connInfo}, nil
}

// buildConnInfo builds a connection string for the database
func buildConnInfo(config config.DatabaseConfig) string {
	connInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.User, config.Password, config.Database, config.SslMode)

	if config.SslMode == "disable" {
		return connInfo
	}

	if config.CACert != "" {
		connInfo += fmt.Sprintf(" sslrootcert=%s", config.CACert)
	}

	if config.ClientCert != "" {
		connInfo += fmt.Sprintf(" sslcert=%s", config.ClientCert)
	}

	if config.ClientKey != "" {
		connInfo += fmt.Sprintf(" sslkey=%s", config.ClientKey)
	}

	return connInfo
}

// checkAndReconnectIfNeeded validates the current connection with a ping
// and tries to reconnect if necessary
func (dbs *SQLdb) checkAndReconnectIfNeeded() {
	start := time.Now()

	for dbs.DB.Ping() != nil {
		log.Errorln("Database unreachable, reconnecting")
		dbs.DB.Close()

		if time.Since(start) > dbReconnectTimeout {
			logFatalf("Could not reconnect to failed database in reasonable time, giving up")
		}
		time.Sleep(dbReconnectSleep)
		log.Debugln("Reconnecting to DB")
		dbs.DB, _ = sqlOpen("postgres", dbs.ConnInfo)
	}

}

// GetFiles retrieves the file details
func (dbs *SQLdb) GetFiles(datasetID string) ([]*FileInfo, error) {
	var (
		r     []*FileInfo = nil
		err   error       = nil
		count int         = 0
	)

	for count < dbRetryTimes {
		r, err = dbs.getFiles(datasetID)
		if err != nil {
			count++
			continue
		}
		break
	}
	return r, err
}

// getFiles is the actual function performing work for GetFile
func (dbs *SQLdb) getFiles(datasetID string) ([]*FileInfo, error) {
	dbs.checkAndReconnectIfNeeded()

	files := []*FileInfo{}
	db := dbs.DB

	const query = "SELECT a.file_id, dataset_id, display_file_name, file_name, file_size, " +
		"decrypted_file_size, decrypted_file_checksum, decrypted_file_checksum_type, file_status from " +
		"local_ega_ebi.file a, local_ega_ebi.file_dataset b WHERE dataset_id = $1 AND a.file_id=b.file_id;"

	rows, err := db.Query(query, datasetID)
	if rows.Err() != nil || err != nil {
		log.Error(err)
		return nil, err
	}
	defer rows.Close()

	// Iterate rows
	for rows.Next() {

		// Read rows into struct
		fi := &FileInfo{}
		err := rows.Scan(&fi.FileID, &fi.DatasetID, &fi.DisplayFileName, &fi.FileName, &fi.FileSize,
			&fi.DecryptedFileSize, &fi.DecryptedFileChecksum, &fi.DecryptedFileChecksumType, &fi.Status)
		if err != nil {
			log.Error(err)
			return nil, err
		}

		// local_ega_ebi.file:file_size is actually the size of the archive file without header
		// so we need to increase the encrypted file size by the length of the header if the user
		// downloaded the files in encrypted format. I set it as 124 which seems to be the default
		// length, but if files can have greater headers, then we can calculate the length with
		// fd := GetFile() --> len(fd.Header)
		fi.FileSize = fi.FileSize + 124

		// Add structs to array
		files = append(files, fi)
	}

	return files, nil
}

// CheckDataset checks if dataset name exists
func (dbs *SQLdb) CheckDataset(dataset string) (bool, error) {
	var (
		r     bool  = false
		err   error = nil
		count int   = 0
	)

	for count < dbRetryTimes {
		r, err = dbs.checkDataset(dataset)
		if err != nil {
			count++
			continue
		}
		break
	}
	return r, err
}

// checkDataset is the actual function performing work for CheckDataset
func (dbs *SQLdb) checkDataset(dataset string) (bool, error) {
	dbs.checkAndReconnectIfNeeded()

	db := dbs.DB
	const query = "SELECT DISTINCT dataset_stable_id FROM local_ega_ebi.filedataset WHERE dataset_stable_id = $1"

	var datasetName string
	if err := db.QueryRow(query, dataset).Scan(&datasetName); err != nil {
		return false, err
	}

	return true, nil
}

// CheckFilePermission checks if user has permissions to access the dataset the file is a part of
func (dbs *SQLdb) CheckFilePermission(fileID string) (string, error) {
	var (
		r     string = ""
		err   error  = nil
		count int    = 0
	)

	for count < dbRetryTimes {
		r, err = dbs.checkFilePermission(fileID)
		if err != nil {
			count++
			continue
		}
		break
	}
	return r, err
}

// checkFilePermission is the actual function performing work for CheckFilePermission
func (dbs *SQLdb) checkFilePermission(fileID string) (string, error) {
	dbs.checkAndReconnectIfNeeded()

	db := dbs.DB
	const query = "SELECT dataset_id FROM local_ega_ebi.file_dataset WHERE file_id = $1"

	var datasetName string
	if err := db.QueryRow(query, fileID).Scan(&datasetName); err != nil {
		return "", err
	}

	return datasetName, nil
}

// FileDownload details are used for downloading a file
type FileDownload struct {
	ArchivePath string
	ArchiveSize int
	Header      []byte
}

// GetFile retrieves the file header
func (dbs *SQLdb) GetFile(fileID string) (*FileDownload, error) {
	var (
		r     *FileDownload = nil
		err   error         = nil
		count int           = 0
	)
	for count < dbRetryTimes {
		r, err = dbs.getFile(fileID)
		if err != nil {
			count++
			continue
		}
		break
	}
	return r, err
}

// getFile is the actual function performing work for GetFile
func (dbs *SQLdb) getFile(fileID string) (*FileDownload, error) {
	dbs.checkAndReconnectIfNeeded()

	db := dbs.DB
	const query = "SELECT file_path, archive_file_size, header FROM local_ega_ebi.file WHERE file_id = $1"

	fd := &FileDownload{}
	var hexString string
	err := db.QueryRow(query, fileID).Scan(&fd.ArchivePath, &fd.ArchiveSize, &hexString)
	if err != nil {
		return nil, err
	}

	fd.Header, err = hex.DecodeString(hexString)
	if err != nil {
		return nil, err
	}

	return fd, nil
}

// Close terminates the connection to the database
func (dbs *SQLdb) Close() {
	db := dbs.DB
	db.Close()
}
