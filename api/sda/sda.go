package sda

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
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

	// get dataset parameter
	dataset := c.Param("dataset")

	if !strings.HasSuffix(dataset, "/files") {
		c.String(http.StatusNotFound, "API path not found, maybe /files is missing")

		return
	}

	// remove / prefix and /files suffix
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

	// Get query params
	qStart := c.DefaultQuery("startCoordinate", "0")
	qEnd := c.DefaultQuery("endCoordinate", "0")

	// Parse and verify coordinates are valid
	start, err := strconv.ParseInt(qStart, 10, 0)

	if err != nil {
		log.Errorf("failed to convert start coordinate %d to integer, %s", start, err)
		c.String(http.StatusBadRequest, "startCoordinate must be an integer")

		return
	}
	end, err := strconv.ParseInt(qEnd, 10, 0)
	if err != nil {
		log.Errorf("failed to convert end coordinate %d to integer, %s", end, err)

		c.String(http.StatusBadRequest, "endCoordinate must be an integer")

		return
	}
	if end < start {
		log.Errorf("endCoordinate=%d must be greater than startCoordinate=%d", end, start)

		c.String(http.StatusBadRequest, "endCoordinate must be greater than startCoordinate")

		return
	}

	contentLength := fileDetails.DecryptedSize
	if c.Param("type") == "encrypted" {
		end = calculateEncryptedEndPosition(start, end, fileDetails)
		contentLength = int(end)
		log.Debug("calculated end to", end)
	}
	if start == 0 && end == 0 {
		c.Header("Content-Length", fmt.Sprint(contentLength))
	} else {
		// Calculate how much we should read (if given)
		togo := end - start
		log.Debug("partial file! set togo to", togo)
		c.Header("Content-Length", fmt.Sprint(togo))
	}

	// Get archive file handle
	file, err := Backend.NewFileReader(fileDetails.ArchivePath)
	if err != nil {
		log.Errorf("could not find archive file %s, %s", fileDetails.ArchivePath, err)
		c.String(http.StatusInternalServerError, "archive error")

		return
	}

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

	// Prepare the file for streaming, encrypted or decrypted
	var encryptedFileReader io.Reader
	var fileStream io.Reader
	hr := bytes.NewReader(fileDetails.Header)
	encryptedFileReader = io.MultiReader(hr, file)

	switch c.Param("type") {
	case "encrypted":
		if start > 0 {
			// unset content-length
			c.Header("Content-Length", "-1")
			log.Errorf("Start coordinate for encrypted files not implemented! %v", start)
			c.String(http.StatusInternalServerError, "an error occurred")

			return
		}
		fileStream = encryptedFileReader

	default:
		c4ghfileStream, err := streaming.NewCrypt4GHReader(encryptedFileReader, *config.Config.App.Crypt4GHKey, nil)
		defer c4ghfileStream.Close()
		if err != nil {
			log.Errorf("could not prepare file for streaming, %s", err)
			c.String(http.StatusInternalServerError, "file stream error")

			return
		}
		err = truncateStream(c4ghfileStream, c.Writer, start)
		fileStream = c4ghfileStream
		if err != nil {
			log.Errorf("error occurred while finding sending start: %v", err)
			c.String(http.StatusInternalServerError, "an error occurred")

			return
		}
	}

	err = sendStream(fileStream, c.Writer, start, end)
	if err != nil {
		log.Errorf("error occurred while sending stream: %v", err)
		c.String(http.StatusInternalServerError, "an error occurred")

		return
	}
}

var truncateStream = func(reader *streaming.Crypt4GHReader, writer http.ResponseWriter, start int64) error {

	if start != 0 {
		// We don't want to read from start, skip ahead to where we should be
		if _, err := reader.Seek(start, 0); err != nil {
			return err
		}
	}
	return nil
}

// used from: https://github.com/neicnordic/crypt4gh/blob/master/examples/reader/main.go#L48C1-L113C1
var sendStream = func(reader io.Reader, writer http.ResponseWriter, start, end int64) error {

	// Calculate how much we should read (if given)
	togo := end - start

	buf := make([]byte, 4096)

	// Loop until we've read what we should (if no/faulty end given, that's EOF)
	for end == 0 || togo > 0 {
		rbuf := buf

		if end != 0 && togo < 4096 {
			// If we don't want to read as much as 4096 bytes
			rbuf = buf[:togo]
		}
		r, err := reader.Read(rbuf)
		togo -= int64(r)

		// Nothing more to read?
		if err == io.EOF && r == 0 {
			// Fall out without error if we had EOF (if we got any data, do one
			// more lap in the loop)
			return nil
		}

		if err != nil && err != io.EOF {
			// An error we want to signal?
			return err
		}

		wbuf := rbuf[:r]
		for len(wbuf) > 0 {
			// Loop until we've written all that we could read,
			// fall out on error
			w, err := writer.Write(wbuf)

			if err != nil {
				return err
			}
			wbuf = wbuf[w:]
		}
	}

	return nil
}

var calculateEncryptedEndPosition = func(start, end int64, fileDetails *database.FileDownload) int64 {
	headlength := bytes.NewReader(fileDetails.Header)
	bodyEnd := int64(fileDetails.ArchiveSize)
	if end > 0 {
		var packageSize float64 = 64000
		togo := end - start
		log.Debug("headlength size: ", headlength.Size())
		bodysize := math.Max(float64(togo-headlength.Size()), 0)
		log.Debug("body size: ", bodysize)
		log.Debug("#packages: ", math.Ceil(bodysize/packageSize))
		endCoord := packageSize * math.Ceil(bodysize/packageSize)
		log.Debug("endCoord: ", endCoord)
		bodyEnd = int64(math.Min(float64(bodyEnd), endCoord))
		log.Debug("body end: ", bodyEnd)
	}
	log.Debug("setting end: ", headlength.Len()+int(bodyEnd))
	return headlength.Size() + bodyEnd
}
