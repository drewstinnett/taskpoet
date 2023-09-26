package taskpoet

import (
	"embed"
	"fmt"
	"log/slog"
	"net/http"
	"path"
	"strconv"

	"github.com/gin-contrib/cors"

	"github.com/gin-gonic/gin"
)

//go:embed static/*
var apiDir embed.FS

// Pagination is the pagination yo
type Pagination struct {
	Limit   uint   `json:"limit"`
	Page    uint   `json:"page"`
	Sort    string `json:"sort"`
	HasMore bool   `json:"hasmore"`
}

// RouterConfig configures the router
type RouterConfig struct {
	Debug       bool
	Port        uint
	LocalClient *Poet
}

// NewRouter returns a new http router
func NewRouter(rc *RouterConfig) *gin.Engine {
	if rc == nil {
		rc = &RouterConfig{}
	}
	if !rc.Debug {
		slog.Warn("Running in release mode")
		gin.SetMode(gin.ReleaseMode)
	} else {
		slog.Warn("Running in debug mode")
		gin.SetMode(gin.DebugMode)
	}

	r := gin.Default()
	r.Use(APIClient(rc.LocalClient))
	r.Use(gin.Recovery())

	if gin.Mode() == "debug" {
		r.Use(cors.Default())
	}

	apiV1 := r.Group("/v1")

	apiV1.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})
	// apiV1.GET("/active", TaskAPIListActive)
	// apiV1.GET("/completed", TaskAPIListCompleted)
	apiV1.GET("/tasks", taskAPIList)
	apiV1.POST("/tasks", taskAPIAdd)
	apiV1.GET("/tasks/:id", taskAPIGet)
	apiV1.PUT("/tasks/:id", taskAPIEdit)
	apiV1.DELETE("/tasks/:id", taskAPIDelete)

	// Swagger/OpenAPI Stuff
	// url := ginSwagger.URL("http://localhost:8080/swagger/doc.json")

	// r.Static("/apidocs/", "api/")
	// index.html won't work with this, use index.htm instead
	// See: https://github.com/gin-gonic/gin/issues/2654
	r.GET("/static/*filepath", func(c *gin.Context) {
		p := path.Join(c.Request.URL.Path)
		c.FileFromFS(p, http.FS(apiDir))
	})
	return r
}

// APIClient is an API Client
func APIClient(client *Poet) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("client", *client)
	}
}

func generatePaginationFromRequest(c *gin.Context) Pagination {
	// Initializing default
	//	var mode string
	limit := 10
	page := 1
	sort := "description"
	query := c.Request.URL.Query()
	for key, value := range query {
		queryValue := value[len(value)-1]
		switch key {
		case "limit":
			limit, _ = strconv.Atoi(queryValue)
		case "page":
			page, _ = strconv.Atoi(queryValue)
		case "sort":
			sort = queryValue
		}
	}
	return Pagination{
		Limit: uint(limit),
		Page:  uint(page),
		Sort:  sort,
	}
}

func getBoolParam(c *gin.Context, paramName string, paramDefault bool) (bool, error) {
	paramGot := c.Query(paramName)
	// If nothing, just use the default
	if paramGot == "" {
		return paramDefault, nil
	}
	paramBool, err := strconv.ParseBool(paramGot)
	if err != nil {
		return false, err
	}
	return paramBool, nil
}

func checkAPIErr(c *gin.Context, err error) {
	if err != nil {
		c.AbortWithStatusJSON(500, map[string]string{"message": fmt.Sprintf("%+v", err)})
	}
}
