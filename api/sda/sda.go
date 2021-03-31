package sda

import (
	"os"
	"strconv"

	"github.com/elixir-oslo/crypt4gh/model/headers"
	"github.com/gofiber/fiber/v2"
	"github.com/neicnordic/sda-download/internal/database"
	"github.com/neicnordic/sda-download/internal/files"
	log "github.com/sirupsen/logrus"
)

// Datasets serves a list of permitted datasets
func Datasets(c *fiber.Ctx) error {
	log.Debugf("request to /metadata/datasets from %s", c.Context().RemoteIP().String())

	return c.JSON(c.Locals("datasets"))
}

// find looks for a dataset name in a list of datasets
func find(datasetID string, datasets []string) bool {
	found := false
	for i := range datasets {
		if datasetID == datasets[i] {
			found = true
			break
		}
	}
	return found
}

// Files serves file metadata
func Files(c *fiber.Ctx, datasetID string) error {
	log.Debugf("request to /metadata/datasets/%s/files from %s", datasetID, c.Context().RemoteIP().String())

	if find(datasetID, c.Locals("datasets").([]string)) {
		// Get file metadata
		files, err := database.DB.GetFiles(datasetID)
		if err != nil {
			// something went wrong with querying or parsing rows
			log.Errorf("database query failed, %s", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}
		return c.JSON(files)
	}

	// no matches in database with given datasetID
	return c.SendStatus(fiber.StatusUnauthorized)
}

// Download serves file contents as bytes
func Download(c *fiber.Ctx, fileID string) error {
	log.Debugf("request to /file/%s from %s", fileID, c.Context().RemoteIP().String())

	// Check user has permissions for this file (as part of a dataset)
	dataset, err := database.DB.CheckFilePermission(fileID)
	if err != nil {
		log.Debugf("requested fileID %s does not exist", fileID)
		return fiber.NewError(401, "no datasets found with that file ID")
	}
	// Get datasets from request context, parsed previously by token middleware
	datasets := c.Locals("datasets").([]string)
	permission := false
	for d := range datasets {
		if datasets[d] == dataset {
			permission = true
			break
		}
	}
	if !permission {
		log.Debugf("user requested to view file %s but does not have permissions for dataset %s", fileID, dataset)
		return fiber.NewError(401, "no permissions to view this file")
	}

	// Get file header
	fileDetails, err := database.DB.GetFile(fileID)
	if err != nil {
		log.Errorf("could not retrieve details for file %s, %s", fileID, err)
		return fiber.NewError(500, "could not retrieve file details")
	}

	// Get archive file handle
	file, err := os.Open(fileDetails.ArchivePath)
	if err != nil {
		log.Errorf("could not find archive file %s, %s", fileDetails.ArchivePath, err)
		return fiber.NewError(500, "could not find archive file")
	}

	// Get coordinates
	qStart := c.Query("startCoordinate")
	qEnd := c.Query("endCoordinate")
	coordinates := &headers.DataEditListHeaderPacket{}
	if len(qStart) > 0 && len(qEnd) > 0 {
		start, err := strconv.ParseUint(qStart, 10, 64)
		if err != nil {
			log.Errorf("failed to convert start coordinate %s to integer, %s", qStart, err)
		}
		end, err := strconv.ParseUint(qEnd, 10, 64)
		if err != nil {
			log.Errorf("failed to convert end coordinate %s to integer, %s", qEnd, err)
		}
		coordinates.NumberLengths = 2
		coordinates.Lengths = []uint64{start, end}
	} else {
		coordinates = nil
	}

	// Get file stream
	fileStream, err := files.StreamFile(fileDetails.Header, file, coordinates)
	if err != nil {
		log.Errorf("could not prepare file for streaming, %s", err)
		return fiber.NewError(500, "failed to prepare a file for streaming")
	}

	return c.SendStream(fileStream)
}
