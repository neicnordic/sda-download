package s3

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/neicnordic/sda-download/api/middleware"
	"github.com/neicnordic/sda-download/internal/session"
	"github.com/neicnordic/sda-download/pkg/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type S3TestSuite struct {
	suite.Suite
}

func (suite *S3TestSuite) SetupTest() {

	// Substitute mock functions
	auth.GetToken = func(header http.Header) (string, int, error) {
		return "token", 200, nil
	}
	auth.GetVisas = func(o auth.OIDCDetails, token string) (*auth.Visas, error) {
		return &auth.Visas{}, nil
	}
	auth.GetPermissions = func(visas auth.Visas) []string {
		return []string{"dataset1", "dataset10", "https://url/dataset"}
	}
	session.NewSessionKey = func() string {
		return "key"
	}
}

func (suite *S3TestSuite) TearDownTest() {
}

func TestS3TestSuite(t *testing.T) {
	suite.Run(t, new(S3TestSuite))
}

func (suite *S3TestSuite) TestParseParams() {

	type paramTest struct {
		Path     string
		Dataset  string
		Filename string
	}

	testParams := []paramTest{
		{Path: "/dataset1", Dataset: "dataset1", Filename: ""},
		{Path: "/dataset10", Dataset: "dataset10", Filename: ""},
		{Path: "/dataset1/dir/file.txt", Dataset: "dataset1", Filename: "dir/file.txt"},
		{Path: "/dataset10/file.txt", Dataset: "dataset10", Filename: "file.txt"},
		{Path: "/https://url/dataset/dir/file.txt", Dataset: "https://url/dataset", Filename: "dir/file.txt"},
		{Path: "/https:/url/dataset/dir/file.txt", Dataset: "https://url/dataset", Filename: "dir/file.txt"},
		{Path: "/https%3A%2F%2Furl%2Fdataset/file.txt", Dataset: "https://url/dataset", Filename: "file.txt"},
		{Path: "/https%3A%2Furl%2Fdataset/file.txt", Dataset: "https://url/dataset", Filename: "file.txt"},
	}

	for _, params := range testParams {

		// response function to check parameter parsing
		testParseParams := func(c *gin.Context) {
			parseParams(c)

			assert.Equal(suite.T(), params.Dataset, c.Param("dataset"), "Failed to parse dataset name")
			assert.Equal(suite.T(), params.Filename, c.Param("filename"), "Failed to parse file name")
			c.AbortWithStatus(http.StatusAccepted)
		}

		// Send a request through the middleware to get datasets, then call the
		// test function to test parameter parsing

		w := httptest.NewRecorder()
		_, router := gin.CreateTestContext(w)
		router.GET("/*path", middleware.TokenMiddleware(), testParseParams)
		router.ServeHTTP(w, httptest.NewRequest("GET", params.Path, nil))

		response := w.Result()
		defer response.Body.Close()

		assert.Equal(suite.T(), http.StatusAccepted, response.StatusCode, "Request failed")

	}

}
