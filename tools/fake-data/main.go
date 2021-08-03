package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"reflect"
	"sort"
	"time"

	"github.com/bxcodec/faker/v3"
	"github.com/drewstinnett/taskpoet/taskpoet"
	"github.com/hako/durafmt"
)

func main() {
	dir, _ := ioutil.TempDir("", "taskpoet-tests")
	dbConfig := &taskpoet.DBConfig{Path: dir}
	_ = taskpoet.InitDB(dbConfig)
	localClient, _ := taskpoet.NewLocalClient(dbConfig)
	n := 1
	for n < 100 {

		desc, err := faker.GetLorem().Sentence(reflect.Value{})
		if err != nil {
			log.Fatal(err)
		}
		t := &taskpoet.Task{
			Description: fmt.Sprintf("%+v", desc),
		}

		log.Println(t)
		found, err := localClient.Task.New(t, nil)
		if err != nil {
			log.Fatal(err)
		}
		log.Println(found)
		n++
	}

	results, _ := taskpoet.QueryTasks(localClient, taskpoet.FilterAllTasks, 0)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Added.Before(results[j].Added)
	})
	now := time.Now()
	for _, task := range results {
		//fmt.Printf("%-50s %s\n", task.Description, task.Due)

		//duration := t.Due - now
		duration := task.Due.Sub(now)
		//hrDuration := taskpoet.HumanizeDuration(duration)
		hrDuration := durafmt.Parse(duration).LimitFirstN(1) // // limit first two parts.
		fmt.Println(task.Description, hrDuration)

	}

}

func CheckErr(err error) {
	log.Fatal(err)
}
