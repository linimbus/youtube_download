package main

import (
	"encoding/json"
	"fmt"
	"github.com/astaxie/beego/logs"
	"io/ioutil"
	"time"
)

type JobVideo struct {
	ItagNo     int
	Format     string
	MimeType   string
	FPS        int
	Wight      int
	Heght      int
	Size       int
	Finished   bool
	FileName   string
}

type Job struct {
	DownLoadDir string
	WebUrl      string
	Reserve     bool

	TotalSize   int64
	Status      string

	Tilte       string
	Author      string
	Duration    time.Duration

	Video       []JobVideo
}

type JobCtrl struct {

	cache []*Job
}

func JobAdd()  {

}

var jobCtrl *JobCtrl

func jobSync()  {
	file := fmt.Sprintf("%s\\job.json", appDataDir())

	var output []Job
	for _, v := range jobCtrl.cache {
		output = append(output, *v)
	}

	value, err := json.Marshal(output)
	if err != nil {
		logs.Error(err.Error())
		return
	}

	err = SaveToFile(file, value)
	if err != nil {
		logs.Error(err.Error())
		return
	}
}

func jobLoad() error {
	file := fmt.Sprintf("%s\\job.json", appDataDir())
	value, err := ioutil.ReadFile(file)
	if err != nil {
		logs.Error(err.Error())
		return nil
	}

	var output []Job
	err = json.Unmarshal(value, &output)
	if err != nil {
		logs.Error(err.Error())
		return err
	}

	for _, v := range output {
		temp := v
		jobCtrl.cache = append(jobCtrl.cache, &temp)
	}

	return nil
}

func JobInit() error {
	jobCtrl = new(JobCtrl)
	jobCtrl.cache = make([]*Job, 0)

	err := jobLoad()
	if err != nil {
		logs.Error(err.Error())
		return err
	}

	return nil
}

