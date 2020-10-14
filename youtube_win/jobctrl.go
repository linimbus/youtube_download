package main

import (
	"encoding/json"
	"fmt"
	"github.com/astaxie/beego/logs"
	"github.com/lixiangyun/youtube_download/youtube"
	"io/ioutil"
	"sync"
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
	Timestamp   string

	DownLoadDir string
	WebUrl      string
	Reserve     bool
	ItagNos     []int
	From        string

	TotalSize   int64
	Status      string

	Tilte       string
	Author      string
	Duration    time.Duration

	Video       []JobVideo
}

type JobCtrl struct {
	sync.RWMutex

	video map[string]*youtube.Video
	cache []*Job
}

func JobAdd(v *youtube.Video, itagno []int, weburl string, reserve bool, downloaddir string) error {
	job := new(Job)
	job.Timestamp = GetTimeStampNumber()
	job.WebUrl = weburl
	job.ItagNos = itagno
	job.DownLoadDir = downloaddir
	job.Reserve = reserve

	jobCtrl.Lock()
	jobCtrl.cache = append(jobCtrl.cache, job)
	jobSync()
	jobCtrl.Unlock()

	return nil
}

var jobCtrl *JobCtrl



func JobStart()  {

}

func JobDel(title string, deleteFile bool) error {
	jobCtrl.Lock()
	defer jobCtrl.Unlock()

	for _, v := range jobCtrl.cache {
		if v.Timestamp == title {
			//
		}
	}
	
	jobSync()
	return nil
}

func job2Item(i int,job *Job) *JobItem {
	return &JobItem{
		Index: i,
		Title: job.Timestamp,
		ProgressRate: 0,
		Speed: 0,
		Size: int(job.TotalSize),
		From: job.From,
		Status: "ready",
	}
}

func jobSyncToConsole()  {
	jobCtrl.RLock()
	defer jobCtrl.RUnlock()

	var output []*JobItem
	maxLen := len(jobCtrl.cache)
	for i := 0; i < maxLen; i++ {
		output = append(output,
			job2Item(i, jobCtrl.cache[maxLen-1-i]),
		)
	}

	JobTalbeUpdate(output)
}

func jobSchedTask()  {
	for  {


		time.Sleep(time.Second)
	}
}

func jobConsoleShow()  {
	for  {
		jobSyncToConsole()
		time.Sleep(2 * time.Second)
	}
}

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

	go jobSchedTask()
	go jobConsoleShow()

	return nil
}

