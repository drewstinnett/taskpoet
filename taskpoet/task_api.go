package taskpoet

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	. "github.com/ahmetb/go-linq/v3" // nolint

	"github.com/gin-gonic/gin"
)

// APITaskResponse is the task response
type APITaskResponse struct {
	Pagination Pagination `json:"pagination"`
	Data       []Task     `json:"data"`
}

func taskAPIAdd(c *gin.Context) {
	client, ok := c.Keys["client"].(Poet)
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
	checkAPIErr(c, err)
	jsonDataS := string(jsonData)

	var tasks []Task
	switch {
	case strings.HasPrefix(jsonDataS, "{"):
		var task Task
		uerr := json.Unmarshal(jsonData, &task)
		checkAPIErr(c, uerr)

		tasks = append(tasks, task)
	case strings.HasPrefix(jsonDataS, "["):
		return
	default:
		c.AbortWithStatusJSON(500, map[string]string{"message": "Could not detect if posted data was array or object"})
	}

	// Add Tasks
	err = client.Task.AddSet(tasks)
	checkAPIErr(c, err)
}

func taskAPIList(c *gin.Context) {
	client, ok := c.Keys["client"].(Poet)
	if !ok {
		c.JSON(500, map[string]string{
			"message": "Could not look up LocalClient in context",
		})
	}

	var tasks []Task

	includeCompleted, err := getBoolParam(c, "include_completed", false)
	checkAPIErr(c, err)

	includeActive, err := getBoolParam(c, "include_active", true)
	checkAPIErr(c, err)

	// Must specify active or completed
	if !includeCompleted && !includeActive {
		c.AbortWithStatusJSON(
			500,
			map[string]string{"message": "Must set either include_completed or include_active to true"})
	}

	if includeActive {
		aTasks, err := client.Task.List("/active")
		checkAPIErr(c, err)
		tasks = append(tasks, aTasks...)
	}

	if includeCompleted {
		cTasks, err := client.Task.List("/completed")
		checkAPIErr(c, err)
		tasks = append(tasks, cTasks...)
	}

	// Do some calculations
	totalTasks := len(tasks)

	pagination := generatePaginationFromRequest(c)
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

func taskAPIGet(c *gin.Context) {
	client, _ := c.Keys["client"].(Poet)

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

func taskAPIEdit(c *gin.Context) {
	client, _ := c.Keys["client"].(Poet)

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
	checkAPIErr(c, err)
	err = json.Unmarshal(jsonData, &editTask)
	checkAPIErr(c, err)

	retTask, err := client.Task.Edit(&editTask)
	checkAPIErr(c, err)

	c.JSON(200, retTask)
}

func taskAPIDelete(c *gin.Context) {
	client, _ := c.Keys["client"].(Poet)

	id := c.Param("id")
	task, err := client.Task.GetWithID(id, "", "")
	if err != nil {
		c.JSON(404, map[string]string{
			"message": fmt.Sprintf("Could not find ID with id: %v", id),
		})
		return
	}

	err = client.Task.Delete(task)
	checkAPIErr(c, err)
	c.JSON(200, "")
}
