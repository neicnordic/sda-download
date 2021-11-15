package sda

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/elixir-oslo/crypt4gh/model/headers"
	"github.com/elixir-oslo/crypt4gh/streaming"
	"github.com/neicnordic/sda-download/api/middleware"
	"github.com/neicnordic/sda-download/internal/database"
	"github.com/neicnordic/sda-download/internal/files"
)

func TestDatasets(t *testing.T) {

	// Save original to-be-mocked functions
	originalGetDatasets := middleware.GetDatasets

	// Substitute mock functions
	middleware.GetDatasets = func(ctx context.Context) []string {
		return []string{"dataset1", "dataset2"}
	}

	// Mock request and response holders
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "https://testing.fi", nil)

	// Test the outcomes of the handler
	Datasets(w, r)
	response := w.Result()
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	expectedStatusCode := 200
	expectedBody := []byte(`["dataset1","dataset2"]` + "\n")

	if response.StatusCode != expectedStatusCode {
		t.Errorf("TestDatasets failed, got %d expected %d", response.StatusCode, expectedStatusCode)
	}
	if !bytes.Equal(body, []byte(expectedBody)) {
		// visual byte comparison in terminal (easier to find string differences)
		t.Error(body)
		t.Error([]byte(expectedBody))
		t.Errorf("TestDatasets failed, got %s expected %s", string(body), string(expectedBody))
	}

	// Return mock functions to originals
	middleware.GetDatasets = originalGetDatasets

}

func TestFind_Found(t *testing.T) {

	// Test case
	datasets := []string{"dataset1", "dataset2", "dataset3"}

	// Run test target
	found := find("dataset2", datasets)

	// Expected results
	expectedFound := true

	if found != expectedFound {
		t.Errorf("TestFind_Found failed, got %t expected %t", found, expectedFound)
	}

}

func TestFind_NotFound(t *testing.T) {

	// Test case
	datasets := []string{"dataset1", "dataset2", "dataset3"}

	// Run test target
	found := find("dataset4", datasets)

	// Expected results
	expectedFound := false

	if found != expectedFound {
		t.Errorf("TestFind_Found failed, got %t expected %t", found, expectedFound)
	}

}

func TestGetFiles_Fail_Database(t *testing.T) {

	// Save original to-be-mocked functions
	originalGetDatasets := middleware.GetDatasets
	originalGetFilesDB := database.GetFiles

	// Substitute mock functions
	middleware.GetDatasets = func(ctx context.Context) []string {
		return []string{"dataset1", "dataset2"}
	}
	database.GetFiles = func(datasetID string) ([]*database.FileInfo, error) {
		return nil, errors.New("something went wrong")
	}

	// Run test target
	fileInfo, statusCode, err := getFiles("dataset1", context.TODO())

	// Expected results
	expectedStatusCode := 500
	expectedError := "database error"

	if fileInfo != nil {
		t.Errorf("TestGetFiles_Fail_Database failed, got %v expected nil", fileInfo)
	}
	if statusCode != expectedStatusCode {
		t.Errorf("TestGetFiles_Fail_Database failed, got %d expected %d", statusCode, expectedStatusCode)
	}
	if err.Error() != expectedError {
		t.Errorf("TestGetFiles_Fail_Database failed, got %v expected %s", err, expectedError)
	}

	// Return mock functions to originals
	middleware.GetDatasets = originalGetDatasets
	database.GetFiles = originalGetFilesDB

}

func TestGetFiles_Fail_NotFound(t *testing.T) {

	// Save original to-be-mocked functions
	originalGetDatasets := middleware.GetDatasets

	// Substitute mock functions
	middleware.GetDatasets = func(ctx context.Context) []string {
		return []string{"dataset1", "dataset2"}
	}

	// Run test target
	fileInfo, statusCode, err := getFiles("dataset3", context.TODO())

	// Expected results
	expectedStatusCode := 404
	expectedError := "dataset not found"

	if fileInfo != nil {
		t.Errorf("TestGetFiles_Fail_NotFound failed, got %v expected nil", fileInfo)
	}
	if statusCode != expectedStatusCode {
		t.Errorf("TestGetFiles_Fail_NotFound failed, got %d expected %d", statusCode, expectedStatusCode)
	}
	if err.Error() != expectedError {
		t.Errorf("TestGetFiles_Fail_NotFound failed, got %v expected %s", err, expectedError)
	}

	// Return mock functions to originals
	middleware.GetDatasets = originalGetDatasets
}

func TestGetFiles_Success(t *testing.T) {

	// Save original to-be-mocked functions
	originalGetDatasets := middleware.GetDatasets
	originalGetFilesDB := database.GetFiles

	// Substitute mock functions
	middleware.GetDatasets = func(ctx context.Context) []string {
		return []string{"dataset1", "dataset2"}
	}
	database.GetFiles = func(datasetID string) ([]*database.FileInfo, error) {
		fileInfo := database.FileInfo{
			FileID: "file1",
		}
		files := []*database.FileInfo{}
		files = append(files, &fileInfo)
		return files, nil
	}

	// Run test target
	fileInfo, statusCode, err := getFiles("dataset1", context.TODO())

	// Expected results
	expectedStatusCode := 200
	expectedFileID := "file1"

	if fileInfo[0].FileID != expectedFileID {
		t.Errorf("TestGetFiles_Success failed, got %v expected nil", fileInfo)
	}
	if statusCode != expectedStatusCode {
		t.Errorf("TestGetFiles_Success failed, got %d expected %d", statusCode, expectedStatusCode)
	}
	if err != nil {
		t.Errorf("TestGetFiles_Success failed, got %v expected nil", err)
	}

	// Return mock functions to originals
	middleware.GetDatasets = originalGetDatasets
	database.GetFiles = originalGetFilesDB

}

func TestFiles_Fail(t *testing.T) {

	// Save original to-be-mocked functions
	originalGetFiles := getFiles

	// Substitute mock functions
	getFiles = func(datasetID string, ctx context.Context) ([]*database.FileInfo, int, error) {
		return nil, 404, errors.New("dataset not found")
	}

	// Mock request and response holders
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "https://testing.fi", nil)

	// Test the outcomes of the handler
	Files(w, r)
	response := w.Result()
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	expectedStatusCode := 404
	expectedBody := []byte("dataset not found\n")

	if response.StatusCode != expectedStatusCode {
		t.Errorf("TestDatasets failed, got %d expected %d", response.StatusCode, expectedStatusCode)
	}
	if !bytes.Equal(body, []byte(expectedBody)) {
		// visual byte comparison in terminal (easier to find string differences)
		t.Error(body)
		t.Error([]byte(expectedBody))
		t.Errorf("TestDatasets failed, got %s expected %s", string(body), string(expectedBody))
	}

	// Return mock functions to originals
	getFiles = originalGetFiles

}

func TestFiles_Success(t *testing.T) {

	// Save original to-be-mocked functions
	originalGetFiles := getFiles

	// Substitute mock functions
	getFiles = func(datasetID string, ctx context.Context) ([]*database.FileInfo, int, error) {
		fileInfo := database.FileInfo{
			FileID:                    "file1",
			DatasetID:                 "dataset1",
			DisplayFileName:           "file1.txt",
			FileName:                  "file1.txt",
			FileSize:                  200,
			DecryptedFileSize:         100,
			DecryptedFileChecksum:     "hash",
			DecryptedFileChecksumType: "sha256",
			Status:                    "READY",
		}
		files := []*database.FileInfo{}
		files = append(files, &fileInfo)
		return files, 200, nil
	}

	// Mock request and response holders
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "https://testing.fi", nil)

	// Test the outcomes of the handler
	Files(w, r)
	response := w.Result()
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	expectedStatusCode := 200
	expectedBody := []byte(
		`[{"fileId":"file1","datasetId":"dataset1","displayFileName":"file1.txt","fileName":` +
			`"file1.txt","fileSize":200,"decryptedFileSize":100,"decryptedFileChecksum":"hash",` +
			`"decryptedFileChecksumType":"sha256","fileStatus":"READY"}]` + "\n")

	if response.StatusCode != expectedStatusCode {
		t.Errorf("TestDatasets failed, got %d expected %d", response.StatusCode, expectedStatusCode)
	}
	if !bytes.Equal(body, []byte(expectedBody)) {
		// visual byte comparison in terminal (easier to find string differences)
		t.Error(body)
		t.Error([]byte(expectedBody))
		t.Errorf("TestDatasets failed, got %s expected %s", string(body), string(expectedBody))
	}

	// Return mock functions to originals
	getFiles = originalGetFiles

}

func TestParseCoordinates_Fail_Start(t *testing.T) {

	// Test case
	// startCoordinate must be an integer
	r := httptest.NewRequest("GET", "https://testing.fi?startCoordinate=x&endCoordinate=100", nil)

	// Run test target
	coordinates, err := parseCoordinates(r)

	// Expected results
	expectedError := "startCoordinate must be an integer"

	if err.Error() != expectedError {
		t.Errorf("TestParseCoordinates_Fail_Start failed, got %s expected %s", err.Error(), expectedError)
	}
	if coordinates != nil {
		t.Errorf("TestParseCoordinates_Fail_Start failed, got %v expected nil", coordinates)
	}

}

func TestParseCoordinates_Fail_End(t *testing.T) {

	// Test case
	// endCoordinate must be an integer
	r := httptest.NewRequest("GET", "https://testing.fi?startCoordinate=0&endCoordinate=y", nil)

	// Run test target
	coordinates, err := parseCoordinates(r)

	// Expected results
	expectedError := "endCoordinate must be an integer"

	if err.Error() != expectedError {
		t.Errorf("TestParseCoordinates_Fail_End failed, got %s expected %s", err.Error(), expectedError)
	}
	if coordinates != nil {
		t.Errorf("TestParseCoordinates_Fail_End failed, got %v expected nil", coordinates)
	}

}

func TestParseCoordinates_Fail_SizeComparison(t *testing.T) {

	// Test case
	// endCoordinate must be greater than startCoordinate
	r := httptest.NewRequest("GET", "https://testing.fi?startCoordinate=50&endCoordinate=100", nil)

	// Run test target
	coordinates, err := parseCoordinates(r)

	// Expected results
	expectedLength := uint32(2)
	expectedStart := uint64(50)
	expectedBytesToRead := uint64(50)

	if err != nil {
		t.Errorf("TestParseCoordinates_Fail_SizeComparison failed, got %v expected nil", err)
	}
	// nolint:staticcheck
	if coordinates == nil {
		t.Error("TestParseCoordinates_Fail_SizeComparison failed, got nil expected not nil")
	}
	// nolint:staticcheck
	if coordinates.NumberLengths != expectedLength {
		t.Errorf("TestParseCoordinates_Fail_SizeComparison failed, got %d expected %d", coordinates.Lengths, expectedLength)
	}
	if coordinates.Lengths[0] != expectedStart {
		t.Errorf("TestParseCoordinates_Fail_SizeComparison failed, got %d expected %d", coordinates.Lengths, expectedLength)
	}
	if coordinates.Lengths[1] != expectedBytesToRead {
		t.Errorf("TestParseCoordinates_Fail_SizeComparison failed, got %d expected %d", coordinates.Lengths, expectedLength)
	}

}

func TestParseCoordinates_Success(t *testing.T) {

	// Test case
	// endCoordinate must be greater than startCoordinate
	r := httptest.NewRequest("GET", "https://testing.fi?startCoordinate=100&endCoordinate=50", nil)

	// Run test target
	coordinates, err := parseCoordinates(r)

	// Expected results
	expectedError := "endCoordinate must be greater than startCoordinate"

	if err.Error() != expectedError {
		t.Errorf("TestParseCoordinates_Fail_SizeComparison failed, got %s expected %s", err.Error(), expectedError)
	}
	if coordinates != nil {
		t.Errorf("TestParseCoordinates_Fail_SizeComparison failed, got %v expected nil", coordinates)
	}

}

func TestDownload_Fail_FileNotFound(t *testing.T) {

	// Save original to-be-mocked functions
	originalCheckFilePermission := database.CheckFilePermission

	// Substitute mock functions
	database.CheckFilePermission = func(fileID string) (string, error) {
		return "", errors.New("file not found")
	}

	// Mock request and response holders
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "https://testing.fi", nil)

	// Test the outcomes of the handler
	Download(w, r)
	response := w.Result()
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	expectedStatusCode := 404
	expectedBody := []byte("file not found\n")

	if response.StatusCode != expectedStatusCode {
		t.Errorf("TestDownload_Fail_FileNotFound failed, got %d expected %d", response.StatusCode, expectedStatusCode)
	}
	if !bytes.Equal(body, []byte(expectedBody)) {
		// visual byte comparison in terminal (easier to find string differences)
		t.Error(body)
		t.Error([]byte(expectedBody))
		t.Errorf("TestDownload_Fail_FileNotFound failed, got %s expected %s", string(body), string(expectedBody))
	}

	// Return mock functions to originals
	database.CheckFilePermission = originalCheckFilePermission

}

func TestDownload_Fail_NoPermissions(t *testing.T) {

	// Save original to-be-mocked functions
	originalCheckFilePermission := database.CheckFilePermission
	originalGetDatasets := middleware.GetDatasets

	// Substitute mock functions
	database.CheckFilePermission = func(fileID string) (string, error) {
		// nolint:goconst
		return "dataset1", nil
	}
	middleware.GetDatasets = func(ctx context.Context) []string {
		return []string{}
	}

	// Mock request and response holders
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "https://testing.fi", nil)

	// Test the outcomes of the handler
	Download(w, r)
	response := w.Result()
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	expectedStatusCode := 401
	expectedBody := []byte("unauthorised\n")

	if response.StatusCode != expectedStatusCode {
		t.Errorf("TestDownload_Fail_NoPermissions failed, got %d expected %d", response.StatusCode, expectedStatusCode)
	}
	if !bytes.Equal(body, []byte(expectedBody)) {
		// visual byte comparison in terminal (easier to find string differences)
		t.Error(body)
		t.Error([]byte(expectedBody))
		t.Errorf("TestDownload_Fail_NoPermissions failed, got %s expected %s", string(body), string(expectedBody))
	}

	// Return mock functions to originals
	database.CheckFilePermission = originalCheckFilePermission
	middleware.GetDatasets = originalGetDatasets

}

func TestDownload_Fail_GetFile(t *testing.T) {

	// Save original to-be-mocked functions
	originalCheckFilePermission := database.CheckFilePermission
	originalGetDatasets := middleware.GetDatasets
	originalGetFile := database.GetFile

	// Substitute mock functions
	database.CheckFilePermission = func(fileID string) (string, error) {
		return "dataset1", nil
	}
	middleware.GetDatasets = func(ctx context.Context) []string {
		return []string{"dataset1"}
	}
	database.GetFile = func(fileID string) (*database.FileDownload, error) {
		return nil, errors.New("database error")
	}

	// Mock request and response holders
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "https://testing.fi", nil)

	// Test the outcomes of the handler
	Download(w, r)
	response := w.Result()
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	expectedStatusCode := 500
	expectedBody := []byte("database error\n")

	if response.StatusCode != expectedStatusCode {
		t.Errorf("TestDownload_Fail_GetFile failed, got %d expected %d", response.StatusCode, expectedStatusCode)
	}
	if !bytes.Equal(body, []byte(expectedBody)) {
		// visual byte comparison in terminal (easier to find string differences)
		t.Error(body)
		t.Error([]byte(expectedBody))
		t.Errorf("TestDownload_Fail_GetFile failed, got %s expected %s", string(body), string(expectedBody))
	}

	// Return mock functions to originals
	database.CheckFilePermission = originalCheckFilePermission
	middleware.GetDatasets = originalGetDatasets
	database.GetFile = originalGetFile

}

func TestDownload_Fail_OpenFile(t *testing.T) {

	// Save original to-be-mocked functions
	originalCheckFilePermission := database.CheckFilePermission
	originalGetDatasets := middleware.GetDatasets
	originalGetFile := database.GetFile

	// Substitute mock functions
	database.CheckFilePermission = func(fileID string) (string, error) {
		return "dataset1", nil
	}
	middleware.GetDatasets = func(ctx context.Context) []string {
		return []string{"dataset1"}
	}
	database.GetFile = func(fileID string) (*database.FileDownload, error) {
		fileDetails := &database.FileDownload{
			ArchivePath: "non-existant-file.txt",
			ArchiveSize: 0,
			Header:      []byte{},
		}
		return fileDetails, nil
	}

	// Mock request and response holders
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "https://testing.fi", nil)

	// Test the outcomes of the handler
	Download(w, r)
	response := w.Result()
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	expectedStatusCode := 500
	expectedBody := []byte("archive error\n")

	if response.StatusCode != expectedStatusCode {
		t.Errorf("TestDownload_Fail_OpenFile failed, got %d expected %d", response.StatusCode, expectedStatusCode)
	}
	if !bytes.Equal(body, []byte(expectedBody)) {
		// visual byte comparison in terminal (easier to find string differences)
		t.Error(body)
		t.Error([]byte(expectedBody))
		t.Errorf("TestDownload_Fail_OpenFile failed, got %s expected %s", string(body), string(expectedBody))
	}

	// Return mock functions to originals
	database.CheckFilePermission = originalCheckFilePermission
	middleware.GetDatasets = originalGetDatasets
	database.GetFile = originalGetFile

}

func TestDownload_Fail_ParseCoordinates(t *testing.T) {

	// Save original to-be-mocked functions
	originalCheckFilePermission := database.CheckFilePermission
	originalGetDatasets := middleware.GetDatasets
	originalGetFile := database.GetFile
	originalParseCoordinates := parseCoordinates

	// Substitute mock functions
	database.CheckFilePermission = func(fileID string) (string, error) {
		return "dataset1", nil
	}
	middleware.GetDatasets = func(ctx context.Context) []string {
		return []string{"dataset1"}
	}
	database.GetFile = func(fileID string) (*database.FileDownload, error) {
		fileDetails := &database.FileDownload{
			ArchivePath: "../../README.md",
			ArchiveSize: 0,
			Header:      []byte{},
		}
		return fileDetails, nil
	}
	parseCoordinates = func(r *http.Request) (*headers.DataEditListHeaderPacket, error) {
		return nil, errors.New("bad params")
	}

	// Mock request and response holders
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "https://testing.fi", nil)

	// Test the outcomes of the handler
	Download(w, r)
	response := w.Result()
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	expectedStatusCode := 400
	expectedBody := []byte("bad params\n")

	if response.StatusCode != expectedStatusCode {
		t.Errorf("TestDownload_Fail_ParseCoordinates failed, got %d expected %d", response.StatusCode, expectedStatusCode)
	}
	if !bytes.Equal(body, []byte(expectedBody)) {
		// visual byte comparison in terminal (easier to find string differences)
		t.Error(body)
		t.Error([]byte(expectedBody))
		t.Errorf("TestDownload_Fail_ParseCoordinates failed, got %s expected %s", string(body), string(expectedBody))
	}

	// Return mock functions to originals
	database.CheckFilePermission = originalCheckFilePermission
	middleware.GetDatasets = originalGetDatasets
	database.GetFile = originalGetFile
	parseCoordinates = originalParseCoordinates

}

func TestDownload_Fail_StreamFile(t *testing.T) {

	// Save original to-be-mocked functions
	originalCheckFilePermission := database.CheckFilePermission
	originalGetDatasets := middleware.GetDatasets
	originalGetFile := database.GetFile
	originalParseCoordinates := parseCoordinates
	originalStreamFile := files.StreamFile

	// Substitute mock functions
	database.CheckFilePermission = func(fileID string) (string, error) {
		return "dataset1", nil
	}
	middleware.GetDatasets = func(ctx context.Context) []string {
		return []string{"dataset1"}
	}
	database.GetFile = func(fileID string) (*database.FileDownload, error) {
		fileDetails := &database.FileDownload{
			ArchivePath: "../../README.md",
			ArchiveSize: 0,
			Header:      []byte{},
		}
		return fileDetails, nil
	}
	parseCoordinates = func(r *http.Request) (*headers.DataEditListHeaderPacket, error) {
		return nil, nil
	}
	files.StreamFile = func(header []byte, file *os.File, coordinates *headers.DataEditListHeaderPacket) (*streaming.Crypt4GHReader, error) {
		return nil, errors.New("file stream error")
	}

	// Mock request and response holders
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "https://testing.fi", nil)

	// Test the outcomes of the handler
	Download(w, r)
	response := w.Result()
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	expectedStatusCode := 500
	expectedBody := []byte("file stream error\n")

	if response.StatusCode != expectedStatusCode {
		t.Errorf("TestDownload_Fail_StreamFile failed, got %d expected %d", response.StatusCode, expectedStatusCode)
	}
	if !bytes.Equal(body, []byte(expectedBody)) {
		// visual byte comparison in terminal (easier to find string differences)
		t.Error(body)
		t.Error([]byte(expectedBody))
		t.Errorf("TestDownload_Fail_StreamFile failed, got %s expected %s", string(body), string(expectedBody))
	}

	// Return mock functions to originals
	database.CheckFilePermission = originalCheckFilePermission
	middleware.GetDatasets = originalGetDatasets
	database.GetFile = originalGetFile
	parseCoordinates = originalParseCoordinates
	files.StreamFile = originalStreamFile

}

func TestDownload_Success(t *testing.T) {

	// Save original to-be-mocked functions
	originalCheckFilePermission := database.CheckFilePermission
	originalGetDatasets := middleware.GetDatasets
	originalGetFile := database.GetFile
	originalParseCoordinates := parseCoordinates
	originalStreamFile := files.StreamFile
	originalSendStream := sendStream

	// Substitute mock functions
	database.CheckFilePermission = func(fileID string) (string, error) {
		return "dataset1", nil
	}
	middleware.GetDatasets = func(ctx context.Context) []string {
		return []string{"dataset1"}
	}
	database.GetFile = func(fileID string) (*database.FileDownload, error) {
		fileDetails := &database.FileDownload{
			ArchivePath: "../../README.md",
			ArchiveSize: 0,
			Header:      []byte{},
		}
		return fileDetails, nil
	}
	parseCoordinates = func(r *http.Request) (*headers.DataEditListHeaderPacket, error) {
		return nil, nil
	}
	files.StreamFile = func(header []byte, file *os.File, coordinates *headers.DataEditListHeaderPacket) (*streaming.Crypt4GHReader, error) {
		return nil, nil
	}
	sendStream = func(w http.ResponseWriter, file io.Reader) {
		fileReader := bytes.NewReader([]byte("hello\n"))
		_, _ = io.Copy(w, fileReader)
	}

	// Mock request and response holders
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "https://testing.fi", nil)

	// Test the outcomes of the handler
	Download(w, r)
	response := w.Result()
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	expectedStatusCode := 200
	expectedBody := []byte("hello\n")

	if response.StatusCode != expectedStatusCode {
		t.Errorf("TestDownload_Success failed, got %d expected %d", response.StatusCode, expectedStatusCode)
	}
	if !bytes.Equal(body, []byte(expectedBody)) {
		// visual byte comparison in terminal (easier to find string differences)
		t.Error(body)
		t.Error([]byte(expectedBody))
		t.Errorf("TestDownload_Success failed, got %s expected %s", string(body), string(expectedBody))
	}

	// Return mock functions to originals
	database.CheckFilePermission = originalCheckFilePermission
	middleware.GetDatasets = originalGetDatasets
	database.GetFile = originalGetFile
	parseCoordinates = originalParseCoordinates
	files.StreamFile = originalStreamFile
	sendStream = originalSendStream

}

func TestSendStream(t *testing.T) {
	// Mock file
	file := []byte("hello\n")
	fileReader := bytes.NewReader(file)

	// Mock stream response
	w := httptest.NewRecorder()
	w.Header().Add("Content-Length", "5")

	// Send file to streamer
	sendStream(w, fileReader)
	response := w.Result()
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	expectedContentLen := "5"
	expectedBody := []byte("hello\n")

	// Verify that stream received contents
	if contentLen := response.Header.Get("Content-Length"); contentLen != expectedContentLen {
		t.Errorf("TestSendStream failed, got %s, expected %s", contentLen, expectedContentLen)
	}
	if !bytes.Equal(body, []byte(expectedBody)) {
		t.Errorf("TestSendStream failed, got %s, expected %s", string(body), string(expectedBody))
	}
}
