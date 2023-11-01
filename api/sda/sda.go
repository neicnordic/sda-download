package sda

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/neicnordic/crypt4gh/model/headers"
	"github.com/neicnordic/crypt4gh/streaming"
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
func Datasets(c *gin.Context) {
	log.Debugf("request permitted datasets")

	// Retrieve dataset list from request context
	// generated by the authentication middleware
	cache := middleware.GetCacheFromContext(c)

	// Return response
	c.JSON(http.StatusOK, cache.Datasets)
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
var getFiles = func(datasetID string, ctx *gin.Context) ([]*database.FileInfo, int, error) {

	// Retrieve dataset list from request context
	// generated by the authentication middleware
	cache := middleware.GetCacheFromContext(ctx)

	log.Debugf("request to process files for dataset %s", sanitizeString(datasetID))

	if find(datasetID, cache.Datasets) {
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
func Files(c *gin.Context) {

	// get dataset parameter, remove / prefix and /files suffix
	dataset := c.Param("dataset")
	dataset = strings.TrimPrefix(dataset, "/")
	dataset = strings.TrimSuffix(dataset, "/files")

	// Get optional dataset scheme
	// A scheme can be delivered separately in a query parameter
	// as schemes may sometimes be problematic when they travel
	// in the path. A client can conveniently split the scheme with "://"
	// which results in 1 item if there is no scheme (e.g. EGAD) or 2 items
	// if there was a scheme (e.g. DOI)
	scheme := c.Query("scheme")
	schemeLogs := strings.ReplaceAll(scheme, "\n", "")
	schemeLogs = strings.ReplaceAll(schemeLogs, "\r", "")

	datasetLogs := strings.ReplaceAll(dataset, "\n", "")
	datasetLogs = strings.ReplaceAll(datasetLogs, "\r", "")
	if scheme != "" {
		log.Debugf("adding scheme=%s to dataset=%s", schemeLogs, datasetLogs)
		dataset = fmt.Sprintf("%s://%s", scheme, dataset)
		log.Debugf("new dataset=%s", datasetLogs)
	}

	// Get dataset files
	files, code, err := getFiles(dataset, c)
	if err != nil {
		c.String(code, err.Error())

		return
	}

	// Return response
	c.JSON(http.StatusOK, files)
}

// Download serves file contents as bytes
func Download(c *gin.Context) {

	// Get file ID from path
	fileID := c.Param("fileid")

	// Check user has permissions for this file (as part of a dataset)
	dataset, err := database.CheckFilePermission(fileID)
	if err != nil {
		c.String(http.StatusNotFound, "file not found")

		return
	}

	// Get datasets from request context, parsed previously by token middleware
	cache := middleware.GetCacheFromContext(c)

	// Verify user has permission to datafile
	permission := false
	for d := range cache.Datasets {
		if cache.Datasets[d] == dataset {
			permission = true

			break
		}
	}
	if !permission {
		log.Debugf("user requested to view file, but does not have permissions for dataset %s", dataset)
		c.String(http.StatusUnauthorized, "unauthorised")

		return
	}

	// Get file header
	fileDetails, err := database.GetFile(fileID)
	if err != nil {
		c.String(http.StatusInternalServerError, "database error")

		return
	}

	// Get archive file handle
	file, err := Backend.NewFileReader(fileDetails.ArchivePath)
	if err != nil {
		log.Errorf("could not find archive file %s, %s", fileDetails.ArchivePath, err)
		c.String(http.StatusInternalServerError, "archive error")

		return
	}

	// Get coordinates
	coordinates, err := parseCoordinates(c.Request)
	if err != nil {
		log.Errorf("parsing of query param coordinates to crypt4gh format failed, reason: %v", err)
		c.String(http.StatusBadRequest, err.Error())

		return
	}

	c.Header("Content-Length", fmt.Sprint(fileDetails.DecryptedSize))
	c.Header("Content-Type", "application/octet-stream")
	if c.GetBool("S3") {
		lastModified, err := time.Parse(time.RFC3339, fileDetails.LastModified)
		if err != nil {
			log.Errorf("failed to parse last modified time: %v", err)
			c.AbortWithStatus(http.StatusInternalServerError)

			return
		}

		c.Header("Content-Disposition", fmt.Sprintf("filename: %v", fileID))
		c.Header("ETag", fileDetails.DecryptedChecksum)
		c.Header("Last-Modified", lastModified.Format(http.TimeFormat))
	}

	if c.Request.Method == http.MethodHead {

		return
	}

	var fileStream io.Reader
	switch c.Param("type") {
	case "encrypted":
		log.Print("Return encrypted file")
		fileStream, err = stitchEncryptedFile(fileDetails.Header, file, coordinates)
		if err != nil {
			log.Errorf("could not prepare file for streaming, %s", err)
			c.String(http.StatusInternalServerError, "file stream error")

			return
		}
		c.Header("Content-Length", "")
	default:
		// Stitch file and prepare it for streaming
		fileStream, err = stitchFile(fileDetails.Header, file, coordinates)
		if err != nil {
			log.Errorf("could not prepare file for streaming, %s", err)
			c.String(http.StatusInternalServerError, "file stream error")

			return
		}
	}

	sendStream(c.Writer, fileStream)
}

// stitchFile stitches the header and file body together for Crypt4GHReader
// and returns a streamable Reader
var stitchEncryptedFile = func(header []byte, file io.ReadCloser, coordinates *headers.DataEditListHeaderPacket) (io.Reader, error) {
	log.Debugf("stitching header to file %s for streaming", file)
	// Stitch header and file body together
	hr := bytes.NewReader(header)

	encryptedFile := io.MultiReader(hr, io.MultiReader(hr, file))

	log.Print("Encrypted file:", encryptedFile)

	log.Debugf("file stream for %s constructed", file)

	return encryptedFile, nil
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

	n, err := io.Copy(w, file)
	log.Debug("end data stream")

	if err != nil {
		log.Errorf("file streaming failed, reason: %v", err)
		http.Error(w, "file streaming failed", 500)

		return
	}

	log.Debugf("Sent %d bytes", n)
}
