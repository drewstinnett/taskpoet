package taskpoet

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	. "github.com/ahmetb/go-linq/v3"

	"github.com/gin-gonic/gin"
)

type APITaskResponse struct {
	Pagination Pagination `json:"pagination"`
	Data       []Task     `json:"data"`
}

func TaskAPIAdd(c *gin.Context) {
	client, ok := c.Keys["client"].(LocalClient)
	if !ok {
		c.JSON(500, map[string]string{
			"message": "Could not look up LocalClient in context",
		})
	}

	if c.Request.Body == nil {
		c.AbortWithStatusJSON(500, map[string]string{"message": "Must post a Task or Tasks json object"})
		return
	}
	jsonData, err := ioutil.ReadAll(c.Request.Body)
	CheckAPIErr(c, err)
	jsonDataS := string(jsonData)

	var tasks []Task
	if strings.HasPrefix(jsonDataS, "{") {
		var task Task
		err := json.Unmarshal(jsonData, &task)
		CheckAPIErr(c, err)

		tasks = append(tasks, task)
	} else if strings.HasPrefix(jsonDataS, "[") {
		return
	} else {
		c.AbortWithStatusJSON(500, map[string]string{"message": "Could not detect if posted data was array or object"})
	}

	// Add Tasks
	err = client.Task.AddSet(tasks, nil)
	CheckAPIErr(c, err)
}

func TaskAPIList(c *gin.Context) {
	client, ok := c.Keys["client"].(LocalClient)
	if !ok {
		c.JSON(500, map[string]string{
			"message": "Could not look up LocalClient in context",
		})
	}

	var tasks []Task

	includeCompleted, err := GetBoolParam(c, "include_completed", false)
	CheckAPIErr(c, err)

	includeActive, err := GetBoolParam(c, "include_active", true)
	CheckAPIErr(c, err)

	// Must specify active or completed
	if !includeCompleted && !includeActive {
		c.AbortWithStatusJSON(
			500,
			map[string]string{"message": fmt.Sprint("Must set either include_completed or include_active to true")})
	}

	if includeActive {
		aTasks, err := client.Task.List("/active")
		CheckAPIErr(c, err)
		tasks = append(tasks, aTasks...)
	}

	if includeCompleted {
		cTasks, err := client.Task.List("/completed")
		CheckAPIErr(c, err)
		tasks = append(tasks, cTasks...)
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

func TaskAPIGet(c *gin.Context) {
	client, _ := c.Keys["client"].(LocalClient)

	id := c.Param("id")
	task, err := client.Task.GetWithID(id, "", "")
	if err != nil {
		c.JSON(404, map[string]string{
			"message": fmt.Sprintf("Could not find ID with id: %v", id),
		})
		return
	}

	c.JSON(200, task)

}

func TaskAPIEdit(c *gin.Context) {
	client, _ := c.Keys["client"].(LocalClient)

	id := c.Param("id")
	_, err := client.Task.GetWithID(id, "", "")
	if err != nil {
		c.JSON(404, map[string]string{
			"message": fmt.Sprintf("Could not find ID with id: %v", id),
		})
		return
	}

	var editTask Task
	jsonData, err := ioutil.ReadAll(c.Request.Body)
	CheckAPIErr(c, err)
	err = json.Unmarshal(jsonData, &editTask)
	CheckAPIErr(c, err)

	retTask, err := client.Task.Edit(&editTask)
	CheckAPIErr(c, err)

	c.JSON(200, retTask)

}

func TaskAPIDelete(c *gin.Context) {
	client, _ := c.Keys["client"].(LocalClient)

	id := c.Param("id")
	task, err := client.Task.GetWithID(id, "", "")
	if err != nil {
		c.JSON(404, map[string]string{
			"message": fmt.Sprintf("Could not find ID with id: %v", id),
		})
		return
	}

	err = client.Task.Delete(task)
	CheckAPIErr(c, err)
	c.JSON(200, "")

}
