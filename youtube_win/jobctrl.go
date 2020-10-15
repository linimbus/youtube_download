package main

import (
	"encoding/json"
	"fmt"
	"github.com/astaxie/beego/logs"
	"github.com/lixiangyun/youtube_download/youtube"
	"io/ioutil"
	"net/url"
	"os"
	"sync"
	"time"
)

type Job struct {
	video       *youtube.Video
	download    *VideoDownload

	Timestamp   string
	WebUrl      string
	Reserve     bool
	ItagNos     []int
	From        string
	OutputDir   string

	TotalSize   int64
	SumSize     int64
	Status      string
	Finished    bool

	Tilte       string
	Author      string
	Duration    time.Duration
}

type JobCtrl struct {
	sync.RWMutex
	cache []*Job
	downs []*Job
	queue chan *Job
}

func parseFrom(link string) string {
	urls, err := url.Parse(link)
	if err != nil {
		logs.Error(err.Error())
		return "youtube.com"
	}
	return urls.Hostname()
}

func videoTotalSize(video *youtube.Video, itagno []int) int64 {
	var size int64
	for _, itag := range itagno {
		for _, format := range video.Formats {
			if format.ItagNo == itag {
				size += int64(StringToInt(format.ContentLength))
				break
			}
		}
	}
	return size
}

func JobAdd(video *youtube.Video, itagno []int, weburl string, reserve bool) error {
	job := new(Job)
	job.Timestamp = GetTimeStampNumber()
	job.WebUrl = weburl
	job.ItagNos = itagno
	job.Reserve = reserve
	job.From = parseFrom(weburl)
	job.video = video
	job.TotalSize = videoTotalSize(video, itagno)
	job.OutputDir = fmt.Sprintf("%s\\%s", BaseSettingGet().HomeDir, job.Timestamp)

	if reserve {
		job.Status = STATUS_RESV
	} else {
		job.Status = STATUS_WAIT
	}

	job.Duration = video.Duration
	job.Author = video.Author
	job.Tilte = video.Title

	jobCtrl.Lock()
	jobCtrl.cache = append(jobCtrl.cache, job)
	if !reserve {
		jobCtrl.queue <- job
	}
	jobSync()
	jobCtrl.Unlock()

	return nil
}

var jobCtrl *JobCtrl


func job2Item(i int, job *Job) *JobItem {
	var speed int64

	dl := job.download
	if dl != nil {
		speed = dl.Stat()
	}
	job.SumSize += speed

	var rate int64
	if job.Finished {
		rate = 100
	} else {
		rate = (job.SumSize * 100) / job.TotalSize
	}

	return &JobItem{
		Index: i,
		Title: job.Timestamp,
		ProgressRate: int(rate),
		Speed: int(speed * 8)/2,
		Size: int(job.TotalSize),
		From: job.From,
		Status: job.Status,
		outputDir: job.OutputDir,
	}
}

func JobDelete(list []string, file bool) error {
	jobCtrl.Lock()
	defer jobCtrl.Unlock()

	for _, v := range list {
		for i, job := range jobCtrl.cache {
			if job.Timestamp == v {
				if file {
					err := os.RemoveAll(job.OutputDir)
					if err != nil {
						logs.Error(err.Error())
					}
				}
				jobCtrl.cache = append(jobCtrl.cache[:i], jobCtrl.cache[i+1:]...)
				break
			}
		}
	}
	jobSync()

	return nil
}

func jobSyncToConsole()  {
	var output []*JobItem

	jobCtrl.RLock()
	maxLen := len(jobCtrl.cache)
	for i := 0; i < maxLen; i++ {
		output = append(output,
			job2Item(i, jobCtrl.cache[maxLen-1-i]),
		)
	}
	jobCtrl.RUnlock()
	var speed int
	for _, v := range output {
		speed += v.Speed
	}
	JobTalbeUpdate(output)
	UpdateStatusFlow(speed)
}

func jobRunning(job *Job)  {
	defer func() {
		jobCtrl.Lock()
		for i, v := range jobCtrl.downs {
			if v == job {
				jobCtrl.downs = append(jobCtrl.downs[:i], jobCtrl.downs[i+1:]...)
				break
			}
		}
		jobCtrl.Unlock()
		jobSync()
	}()

	job.SumSize = 0

	var err error
	job.download, err = NewVideoDownload(job.video, job.WebUrl, job.OutputDir, job.ItagNos )
	if err != nil {
		logs.Error(err.Error())
		job.Status = STATUS_STOP
		return
	}

	logs.Info("video download task add: %s", job.WebUrl)

	job.Status = STATUS_LOAD
	job.download.Wait()
	job.Status = STATUS_DONE
	job.Finished = true
}

func jobSchedTask() {
	for  {
		time.Sleep(2 * time.Second)

		cfg := BaseSettingGet()

		jobCtrl.Lock()
		if cfg.MultiThreaded == len(jobCtrl.downs) {
			jobCtrl.Unlock()
			continue
		}

		if cfg.MultiThreaded < len(jobCtrl.downs) {
			// shutdown....
			jobCtrl.Unlock()
			continue
		}

		addnums := cfg.MultiThreaded - len(jobCtrl.downs)
		for i := 0 ; i < addnums; i++ {
			if len(jobCtrl.queue) == 0 {
				break
			}

			addJob := <- jobCtrl.queue
			jobCtrl.downs = append(jobCtrl.downs, addJob)

			go jobRunning(addJob)
		}
		jobCtrl.Unlock()
	}
}

func jobConsoleShow()  {
	for  {
		jobSyncToConsole()
		jobSync()
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

	for _, v := range jobCtrl.cache {
		if v.Finished == false && v.Reserve == false {
			jobCtrl.queue <- v
		}
	}

	return nil
}

func JobInit() error {
	jobCtrl = new(JobCtrl)
	jobCtrl.cache = make([]*Job, 0)
	jobCtrl.queue = make(chan *Job, 1024)
	jobCtrl.downs = make([]*Job, 0)

	err := jobLoad()
	if err != nil {
		logs.Error(err.Error())
		return err
	}

	go jobSchedTask()
	go jobConsoleShow()

	return nil
}

