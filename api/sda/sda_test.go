package sda

import (
	"bytes"
	"context"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/neicnordic/sda-download/api/middleware"
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

	datasets := []string{"dataset1", "dataset2", "dataset3"}

	found := find("dataset2", datasets)

	expectedFound := true

	if found != expectedFound {
		t.Errorf("TestFind_Found failed, got %t expected %t", found, expectedFound)
	}

}

func TestFind_NotFound(t *testing.T) {

	datasets := []string{"dataset1", "dataset2", "dataset3"}

	found := find("dataset4", datasets)

	expectedFound := false

	if found != expectedFound {
		t.Errorf("TestFind_Found failed, got %t expected %t", found, expectedFound)
	}

}
