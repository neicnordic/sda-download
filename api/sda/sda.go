package sda

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/elixir-oslo/crypt4gh/model/headers"
	"github.com/gorilla/mux"
	"github.com/neicnordic/sda-download/api/middleware"
	"github.com/neicnordic/sda-download/internal/config"
	"github.com/neicnordic/sda-download/internal/database"
	"github.com/neicnordic/sda-download/internal/files"
	log "github.com/sirupsen/logrus"
)

// Datasets serves a list of permitted datasets
func Datasets(w http.ResponseWriter, r *http.Request) {
	log.Infof("request to %s", r.URL.Path)

	// Retrieve dataset list from request context
	// generated by the authentication middleware
	datasets := middleware.GetDatasets(r.Context())

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(datasets)
}

// find looks for a dataset name in a list of datasets
func find(datasetID string, datasets []string) bool {
	found := false
	for _, dataset := range datasets {
		if datasetID == dataset {
			found = true
			break
		}
	}
	return found
}

// getFiles returns files belonging to a dataset
func getFiles(datasetID string, ctx context.Context) ([]*database.FileInfo, int, error) {

	// Retrieve dataset list from request context
	// generated by the authentication middleware
	datasets := middleware.GetDatasets(ctx)

	if find(datasetID, datasets) {
		// Get file metadata
		files, err := database.DB.GetFiles(datasetID)
		if err != nil {
			// something went wrong with querying or parsing rows
			log.Errorf("database query failed, %s", err)
			return nil, 500, errors.New("database error")
		}

		return files, 200, nil
	}

	return nil, 404, errors.New("dataset not found")
}

// Files serves a list of files belonging to a dataset
func Files(w http.ResponseWriter, r *http.Request) {
	log.Infof("request to %s", r.URL.Path)
	vars := mux.Vars(r)

	// Get dataset files
	files, code, err := getFiles(vars["dataset"], r.Context())
	if err != nil {
		http.Error(w, err.Error(), code)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(files)
}

// Download serves file contents as bytes
func Download(w http.ResponseWriter, r *http.Request) {
	log.Infof("request to %s", r.URL.Path)

	// Get file ID from path
	fileID := strings.Replace(r.URL.Path, "/files/", "", 1)

	// Check user has permissions for this file (as part of a dataset)
	dataset, err := database.DB.CheckFilePermission(fileID)
	if err != nil {
		log.Debugf("requested fileID %s does not exist", fileID)
		http.Error(w, "file not found", 404)
		return
	}

	// Get datasets from request context, parsed previously by token middleware
	datasets := middleware.GetDatasets(r.Context())

	// Verify user has permission to datafile
	permission := false
	for d := range datasets {
		if datasets[d] == dataset || "https://"+datasets[d] == dataset {
			permission = true
			break
		}
	}
	if !permission {
		log.Debugf("user requested to view file %s but does not have permissions for dataset %s", fileID, dataset)
		http.Error(w, "unauthorised", 401)
		return
	}

	// Get file header
	fileDetails, err := database.DB.GetFile(fileID)
	if err != nil {
		log.Errorf("could not retrieve details for file %s, %s", fileID, err)
		http.Error(w, "database error", 500)
		return
	}

	// Get archive file handle
	path := filepath.Join(config.Config.App.ArchivePath, fileDetails.ArchivePath)
	file, err := os.Open(path)
	if err != nil {
		log.Errorf("could not find archive file %s, %s", fileDetails.ArchivePath, err)
		http.Error(w, "archive error", 500)
		return
	}

	// Get coordinates
	qStart := r.URL.Query().Get("startCoordinate")
	qEnd := r.URL.Query().Get("endCoordinate")
	coordinates := &headers.DataEditListHeaderPacket{}
	if len(qStart) > 0 && len(qEnd) > 0 {
		start, err := strconv.ParseUint(qStart, 10, 64)
		if err != nil {
			log.Errorf("failed to convert start coordinate %s to integer, %s", qStart, err)
			http.Error(w, "startCoordinate must be an integer", 400)
			return
		}
		end, err := strconv.ParseUint(qEnd, 10, 64)
		if err != nil {
			log.Errorf("failed to convert end coordinate %s to integer, %s", qEnd, err)
			http.Error(w, "endCoordinate must be an integer", 400)
			return
		}
		if end < start {
			log.Errorf("endCoordinate=%d must be greater than startCoordinate=%d", end, start)
			http.Error(w, "endCoordinate must be greater than startCoordinate", 400)
			return
		}
		// API query params take a coordinate range to read "start...end"
		// But Crypt4GHReader takes a start byte and number of bytes to read "start...(end-start)"
		bytesToRead := end - start
		coordinates.NumberLengths = 2
		coordinates.Lengths = []uint64{start, bytesToRead}
	} else {
		coordinates = nil
	}

	// Get file stream
	fileStream, err := files.StreamFile(fileDetails.Header, file, coordinates)
	if err != nil {
		log.Errorf("could not prepare file for streaming, %s", err)
		http.Error(w, "file stream error", 500)
		return
	}

	sendStream(w, fileStream)
}

// sendStream streams file contents from a reader
func sendStream(w http.ResponseWriter, file io.Reader) {
	log.Debug("begin data stream")

	w.Header().Set("Content-Type", "application/octet-stream")
	n, err := io.Copy(w, file)
	log.Debug("end data stream")

	if err != nil {
		log.Errorf("file streaming failed, reason: %v", err)
		http.Error(w, "file streaming failed", 500)
		return
	}

	log.Debugf("Sent %d bytes", n)
}
