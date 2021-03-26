package sda

import (
	"bytes"
	"io"
	"os"
	"strconv"

	"github.com/elixir-oslo/crypt4gh/model/headers"
	"github.com/elixir-oslo/crypt4gh/streaming"
	"github.com/gofiber/fiber/v2"
	"github.com/neicnordic/sda-download/internal/config"
	"github.com/neicnordic/sda-download/internal/database"
	"github.com/neicnordic/sda-download/pkg/auth"
	log "github.com/sirupsen/logrus"
)

// Datasets serves a list of permitted datasets
func Datasets(c *fiber.Ctx) error {

	// Get access token
	token, errorCode := auth.GetToken(c.Get("Authorization"))
	if errorCode != 0 {
		log.Debugf("request rejected, %s", token) // contains error message
		return fiber.NewError(errorCode, token)
	}

	// Get permissions
	visas := c.Locals("visas")
	datasets := auth.GetPermissions(visas.([]byte))
	if len(datasets) == 0 {
		log.Debug("token carries no dataset permissions matching the database")
		return fiber.NewError(404, "no datasets found")
	}

	return c.JSON(datasets)
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

	// Get access token
	token, errorCode := auth.GetToken(c.Get("Authorization"))
	if errorCode != 0 {
		log.Debugf("request rejected, %s", token) // contains error message
		return fiber.NewError(errorCode, token)
	}

	// Get permissions
	visas := c.Locals("visas")
	datasets := auth.GetPermissions(visas.([]byte))
	if len(datasets) == 0 {
		log.Debug("token carries no dataset permissions matching the database")
		return fiber.NewError(404, "no datasets found")
	}

	if find(datasetID, datasets) {
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

	// Get access token
	token, errorCode := auth.GetToken(c.Get("Authorization"))
	if errorCode != 0 {
		log.Debugf("request rejected, %s", token) // contains error message
		return fiber.NewError(errorCode, token)
	}

	// Get permissions
	visas := c.Locals("visas")
	datasets := auth.GetPermissions(visas.([]byte))
	if len(datasets) == 0 {
		log.Debug("token carries no dataset permissions matching the database")
		return fiber.NewError(404, "no datasets found")
	}

	// Check user has permissions for this file (as part of a dataset)
	dataset, err := database.DB.CheckFilePermission(fileID)
	if err != nil {
		log.Debugf("requested fileID %s does not exist", fileID)
		return fiber.NewError(401, "no datasets found with that file ID")
	}
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

	// Stitch header and file body together
	hr := bytes.NewReader(fileDetails.Header)
	mr := io.MultiReader(hr, file)
	c4ghr, err := streaming.NewCrypt4GHReader(mr, *config.Config.App.Crypt4GHKey, coordinates)
	if err != nil {
		log.Errorf("failed to create Crypt4GH stream reader, %s", err)
	}

	return c.SendStream(c4ghr)
}
