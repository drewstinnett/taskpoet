package taskpoet

import (
	"fmt"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

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
	apiV1 := r.Group("/v1")

	apiV1.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})
	apiV1.GET("/active", APIActive)

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

	c.JSON(200, tasks)

}

func APIClient(client *LocalClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("client", *client)
	}
}
