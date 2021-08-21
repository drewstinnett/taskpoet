package taskpoet

import (
	"embed"
	"fmt"
	"net/http"
	"path"
	"strconv"

	"github.com/gin-contrib/cors"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

//go:embed static/*
var apiDir embed.FS

type Pagination struct {
	Limit   uint   `json:"limit"`
	Page    uint   `json:"page"`
	Sort    string `json:"sort"`
	HasMore bool   `json:"hasmore"`
}

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
	if gin.Mode() == "debug" {
		r.Use(cors.Default())
	}

	apiV1 := r.Group("/v1")

	apiV1.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})
	//apiV1.GET("/active", TaskAPIListActive)
	//apiV1.GET("/completed", TaskAPIListCompleted)
	apiV1.GET("/tasks", TaskAPIList)
	apiV1.POST("/tasks", TaskAPIAdd)
	apiV1.GET("/tasks/:id", TaskAPIGet)
	apiV1.PUT("/tasks/:id", TaskAPIEdit)
	apiV1.DELETE("/tasks/:id", TaskAPIDelete)

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

func GetBoolParam(c *gin.Context, paramName string, paramDefault bool) (bool, error) {

	paramGot := c.Query(paramName)
	// If nothing, just use the default
	if paramGot == "" {
		return paramDefault, nil
	}
	paramBool, err := strconv.ParseBool(paramGot)
	if err != nil {
		return false, err
	} else {
		return paramBool, nil
	}

}

func CheckAPIErr(c *gin.Context, err error) {
	if err != nil {
		c.AbortWithStatusJSON(500, map[string]string{"message": fmt.Sprintf("%+v", err)})
	}
}
