package sda

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strconv"

	"github.com/elixir-oslo/crypt4gh/model/headers"
	"github.com/elixir-oslo/crypt4gh/streaming"
	"github.com/gorilla/mux"
	"github.com/neicnordic/sda-download/api/middleware"
	"github.com/neicnordic/sda-download/internal/config"
	"github.com/neicnordic/sda-download/internal/database"
	"github.com/neicnordic/sda-download/internal/storage"
	log "github.com/sirupsen/logrus"
)

var Backend storage.Backend

func sanitizeString(str string) string {
	var pattern = regexp.MustCompile(`(https?://[^\s/$.?#].[^\s]+|[A-Za-z0-9-_:.]+)`)
	return pattern.ReplaceAllString(str, "[identifier]: $1")
}

// Datasets serves a list of permitted datasets
func Datasets(w http.ResponseWriter, r *http.Request) {
	log.Debugf("request permitted datasets")

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
var getFiles = func(datasetID string, ctx context.Context) ([]*database.FileInfo, int, error) {

	// Retrieve dataset list from request context
	// generated by the authentication middleware
	datasets := middleware.GetDatasets(ctx)

	log.Debugf("request to process files for dataset %s", sanitizeString(datasetID))

	if find(datasetID, datasets) {
		// Get file metadata
		files, err := database.GetFiles(datasetID)
		if err != nil {
			// something went wrong with querying or parsing rows
			log.Errorf("database query failed for dataset %s, reason %s", sanitizeString(datasetID), err)
			return nil, 500, errors.New("database error")
		}

		return files, 200, nil
	}

	return nil, 404, errors.New("dataset not found")
}

// Files serves a list of files belonging to a dataset
func Files(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	dataset := vars["dataset"]

	// Get dataset files
	files, code, err := getFiles(dataset, r.Context())
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

	// Get file ID from path
	vars := mux.Vars(r)
	fileID := vars["fileid"]

	// Check user has permissions for this file (as part of a dataset)
	dataset, err := database.CheckFilePermission(fileID)
	if err != nil {
		http.Error(w, "file not found", 404)
		return
	}

	// Get datasets from request context, parsed previously by token middleware
	datasets := middleware.GetDatasets(r.Context())

	// Verify user has permission to datafile
	permission := false
	for d := range datasets {
		if datasets[d] == dataset {
			permission = true
			break
		}
	}
	if !permission {
		log.Debugf("user requested to view file, but does not have permissions for dataset %s", dataset)
		http.Error(w, "unauthorised", 401)
		return
	}

	// Get file header
	fileDetails, err := database.GetFile(fileID)
	if err != nil {
		http.Error(w, "database error", 500)
		return
	}

	// Get archive file handle
	file, err := Backend.NewFileReader(fileDetails.ArchivePath)
	if err != nil {
		log.Errorf("could not find archive file %s, %s", fileDetails.ArchivePath, err)
		http.Error(w, "archive error", 500)
		return
	}

	// Get coordinates
	coordinates, err := parseCoordinates(r)
	if err != nil {
		log.Errorf("parsing of query param coordinates to crypt4gh format failed, reason: %v", err)
		http.Error(w, err.Error(), 400)
		return
	}

	// Stitch file and prepare it for streaming
	fileStream, err := stitchFile(fileDetails.Header, file, coordinates)
	if err != nil {
		log.Errorf("could not prepare file for streaming, %s", err)
		http.Error(w, "file stream error", 500)
		return
	}

	sendStream(w, fileStream)
}

// stitchFile stitches the header and file body together for Crypt4GHReader
// and returns a streamable Reader
var stitchFile = func(header []byte, file io.ReadCloser, coordinates *headers.DataEditListHeaderPacket) (*streaming.Crypt4GHReader, error) {
	log.Debugf("stitching header to file %s for streaming", file)
	// Stitch header and file body together
	hr := bytes.NewReader(header)
	mr := io.MultiReader(hr, file)
	c4ghr, err := streaming.NewCrypt4GHReader(mr, *config.Config.App.Crypt4GHKey, coordinates)
	if err != nil {
		log.Errorf("failed to create Crypt4GH stream reader, %v", err)
		return nil, err
	}
	log.Debugf("file stream for %s constructed", file)
	return c4ghr, nil
}

// parseCoordinates takes query param coordinates and converts them to
// Crypt4GH reader format
var parseCoordinates = func(r *http.Request) (*headers.DataEditListHeaderPacket, error) {

	coordinates := &headers.DataEditListHeaderPacket{}

	// Get query params
	qStart := r.URL.Query().Get("startCoordinate")
	qEnd := r.URL.Query().Get("endCoordinate")

	// Parse and verify coordinates are valid
	if len(qStart) > 0 && len(qEnd) > 0 {
		start, err := strconv.ParseUint(qStart, 10, 64)
		if err != nil {
			log.Errorf("failed to convert start coordinate %d to integer, %s", start, err)
			return nil, errors.New("startCoordinate must be an integer")
		}
		end, err := strconv.ParseUint(qEnd, 10, 64)
		if err != nil {
			log.Errorf("failed to convert end coordinate %d to integer, %s", end, err)
			return nil, errors.New("endCoordinate must be an integer")
		}
		if end < start {
			log.Errorf("endCoordinate=%d must be greater than startCoordinate=%d", end, start)
			return nil, errors.New("endCoordinate must be greater than startCoordinate")
		}
		// API query params take a coordinate range to read "start...end"
		// But Crypt4GHReader takes a start byte and number of bytes to read "start...(end-start)"
		bytesToRead := end - start
		coordinates.NumberLengths = 2
		coordinates.Lengths = []uint64{start, bytesToRead}
	} else {
		coordinates = nil
	}

	return coordinates, nil
}

// sendStream streams file contents from a reader
var sendStream = func(w http.ResponseWriter, file io.Reader) {
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
