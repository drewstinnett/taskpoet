package taskpoet

import (
	"embed"
	"fmt"
	"net/http"
	"path"
	"strconv"

	. "github.com/ahmetb/go-linq/v3"
	"github.com/gin-contrib/cors"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

//go:embed static/*
var apiDir embed.FS

type RouterConfig struct {
	Debug       bool
	Port        uint
	LocalClient *LocalClient
}

func NewRouter(rc *RouterConfig) *gin.Engine {
	if rc == nil {
		rc = &RouterConfig{}
	}
	if !rc.Debug {
		log.Warning("Running in release mode")
		gin.SetMode(gin.ReleaseMode)
	} else {
		log.Warning("Running in debug mode")
		gin.SetMode(gin.DebugMode)
	}

	r := gin.Default()
	r.Use(APIClient(rc.LocalClient))
	r.Use(gin.Recovery())

	// TODO: Explore what this should actually be
	if gin.Mode() == "default" {
		r.Use(cors.Default())
	}

	apiV1 := r.Group("/v1")

	apiV1.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})
	apiV1.GET("/active", APIActive)

	// Swagger/OpenAPI Stuff
	//url := ginSwagger.URL("http://localhost:8080/swagger/doc.json")

	// r.Static("/apidocs/", "api/")
	// index.html won't work with this, use index.htm instead
	// See: https://github.com/gin-gonic/gin/issues/2654
	r.GET("/static/*filepath", func(c *gin.Context) {
		p := path.Join(c.Request.URL.Path)
		c.FileFromFS(p, http.FS(apiDir))
	})
	return r
}

func APIActive(c *gin.Context) {
	client, ok := c.Keys["client"].(LocalClient)
	if !ok {
		c.JSON(500, map[string]string{
			"message": "Could not look up LocalClient in context",
		})
	}
	tasks, err := client.Task.List("/active")
	if err != nil {
		c.JSON(500, map[string]string{
			"message": fmt.Sprintf("%+v", err),
		})
	}

	// Do some calculations
	totalTasks := len(tasks)

	pagination := GeneratePaginationFromRequest(c)
	var pageData []Task
	skip := int(pagination.Limit * (pagination.Page - 1))
	From(tasks).Skip(skip).Take(int(pagination.Limit)).ToSlice(&pageData)
	currentMaxTask := skip + len(pageData)

	if currentMaxTask < totalTasks {
		pagination.HasMore = true
	} else {
		pagination.HasMore = false
	}

	c.JSON(200, APITaskResponse{
		Data:       pageData,
		Pagination: pagination,
	})

}

func APIClient(client *LocalClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("client", *client)
	}
}

func GeneratePaginationFromRequest(c *gin.Context) Pagination {
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
			break
		case "page":
			page, _ = strconv.Atoi(queryValue)
			break
		case "sort":
			sort = queryValue
			break

		}
	}
	return Pagination{
		Limit: uint(limit),
		Page:  uint(page),
		Sort:  sort,
	}

}

type Pagination struct {
	Limit   uint   `json:"limit"`
	Page    uint   `json:"page"`
	Sort    string `json:"sort"`
	HasMore bool   `json:"hasmore"`
}

type APITaskResponse struct {
	Pagination Pagination `json:"pagination"`
	Data       []Task     `json:"data"`
}
