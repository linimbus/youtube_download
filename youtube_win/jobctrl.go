package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/astaxie/beego/logs"
	"github.com/lixiangyun/youtube_download/youtube"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

type FormatInfo struct {
	ItagNo    int
	Url       string
	MimeType  string
	Quality   string
	FPS, Width,  Height int
	ContentLength string
	LastModified  string
}

type VideoInfo struct {
	WebUrl   string
	Title    string
	Author   string
	Duration time.Duration
	Formats  []FormatInfo
}

func formatInfoOutput(format youtube.Format) FormatInfo {
	return FormatInfo{
		ItagNo: format.ItagNo,
		Url: format.URL,
		MimeType: format.MimeType,
		Quality: format.Quality,
		FPS: format.FPS, Width: format.Width, Height: format.Height,
		ContentLength: format.ContentLength,
		LastModified: format.LastModified,
	}
}

func videoInfoOutput(weburl string, v *youtube.Video, formats []youtube.Format) *VideoInfo {
	var fmtInfo []FormatInfo
	for _, v := range formats {
		fmtInfo = append(fmtInfo, formatInfoOutput(v))
	}
	return &VideoInfo{WebUrl: weburl,
		Title: v.Title,
		Author: v.Author,
		Duration: v.Duration,
		Formats: fmtInfo,
	}
}

func videoInfomationSave(weburl string, v *youtube.Video, formats []youtube.Format, outputdir string)  {
	value, err := yaml.Marshal(videoInfoOutput(weburl, v, formats))
	if err != nil {
		logs.Error(err.Error())
		return
	}
	filepath := fmt.Sprintf("%s\\info.txt", outputdir)
	err = SaveToFile(filepath, value)
	if err != nil {
		logs.Error(err.Error())
		return
	}
}

type Job struct {
	video       *youtube.Video
	download    *DownloadJob

	Timestamp   string
	WebUrl      string
	Reserve     bool
	From        string
	OutputDir   string
	FileList    []DownLoadFile

	lastSize    int64

	TotalSize   int64
	Status      string
	Finished    bool

	Tilte       string
	Author      string
	Duration    time.Duration
}

type JobCtrl struct {
	sync.Mutex

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

func videoContentLangthGet(video *youtube.Video, format *youtube.Format) int64 {
	httpclient, err := HttpClientGet(HttpProxyGet())
	if err != nil {
		logs.Error(err.Error())
		return 0
	}
	client := &youtube.Client{HTTPClient: httpclient.cli}
	for i:=0; i<5; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 10 * time.Second)
		length, err := client.GetStreamContextLangth(ctx, video, format)
		cancel()
		if err != nil {
			logs.Error(err.Error())
			continue
		}
		return length
	}
	return 0
}

func videoFormatFileName(f *youtube.Format) string {
	values := strings.Split(f.MimeType, ";")
	values = strings.Split(values[0],"/")

	var suffix = "mp4"
	if len(values) == 2 {
		suffix = values[1]
	}
	formatType := values[0]

	if strings.ToLower(formatType) == "audio" {
		return fmt.Sprintf("audio_%d.m4a", f.ItagNo)
	} else {
		return fmt.Sprintf("%s_%d_%dp.%s", formatType, f.ItagNo, f.Height, suffix)
	}
}

func JobAdd(video *youtube.Video, itagno []int, weburl string, reserve bool) error {

	job := new(Job)
	job.Timestamp = GetTimeStampNumber()
	job.WebUrl = weburl
	job.Reserve = reserve
	job.From = parseFrom(weburl)
	job.video = video
	job.OutputDir = fmt.Sprintf("%s\\%s", BaseSettingGet().HomeDir, job.Timestamp)

	err := TouchDir(job.OutputDir)
	if err != nil {
		logs.Error(err.Error())
		return err
	}

	var fileList []DownLoadFile
	var totalSize int64

	for _, v := range itagno {
		for _, format := range video.Formats {
			if v == format.ItagNo {
				contentLength := videoContentLangthGet(video, &format)
				totalSize += contentLength
				fileList = append(fileList, DownLoadFile{
					ItagNo: v,
					TotalSize: contentLength,
					Filepath: fmt.Sprintf("%s\\%s", job.OutputDir, videoFormatFileName(&format)),
				})
			}
		}
	}

	videoInfomationSave(weburl, job.video, video.Formats, job.OutputDir)

	job.TotalSize = totalSize
	job.FileList = fileList

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
		jobToQueue(job)
	}
	jobSync()
	jobCtrl.Unlock()

	return nil
}

var jobCtrl *JobCtrl

func job2Item(i int, job *Job) *JobItem {
	var speed int64

	var sumsize int64
	for _, v := range job.FileList {
		sumsize += v.CurSize
	}

	speed = sumsize - job.lastSize
	job.lastSize = sumsize

	rate := int64(100)
	if !job.Finished {
		rate = (sumsize * 100) / job.TotalSize
		if rate > 100 {
			rate = 99
		}
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

func RemoveAllFile(path string)  {
	Separator := fmt.Sprintf("%c",os.PathSeparator)
	files, err := ioutil.ReadDir(path)
	if err != nil {
		logs.Error(err.Error())
		return
	}
	for _, file := range files {
		filepath := path + Separator + file.Name()
		if file.IsDir() {
			RemoveAllFile(filepath)
		} else {
			err = os.Remove(filepath)
			if err != nil {
				logs.Error(err.Error())
			}
		}
	}
	err = os.RemoveAll(path)
	if err != nil {
		logs.Error(err.Error())
	}
}

func JobDelete(list []string, file bool) error {
	jobCtrl.Lock()
	defer jobCtrl.Unlock()

	for _, v := range list {
		for i, job := range jobCtrl.downs {
			if job.Timestamp == v {
				dl := job.download
				if dl != nil {
					dl.Cancel()
				}
				jobCtrl.downs = append(jobCtrl.downs[:i], jobCtrl.downs[i+1:]...)
				break
			}
		}
	}

	for _, v := range list {
		for i, job := range jobCtrl.cache {
			if job.Timestamp == v {
				if file {
					RemoveAllFile(job.OutputDir)
				}
				job.Status = STATUS_DEL
				jobCtrl.cache = append(jobCtrl.cache[:i], jobCtrl.cache[i+1:]...)
				break
			}
		}
	}
	jobSync()

	logs.Info("job %v delete success", list)

	return nil
}

func jobSyncToConsole()  {
	var output []*JobItem

	jobCtrl.Lock()
	maxLen := len(jobCtrl.cache)
	for i := 0; i < maxLen; i++ {
		output = append(output,
			job2Item(i, jobCtrl.cache[maxLen-1-i]),
		)
	}
	jobSync()
	jobCtrl.Unlock()

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
		jobSync()
		jobCtrl.Unlock()
	}()

	var err error
	job.download, err = NewDownloadJob(job.Timestamp, job.video, job.WebUrl, job.FileList )
	if err != nil {
		logs.Error(err.Error())
		job.Status = STATUS_STOP
		return
	}

	logs.Info("video download task add: %s", job.WebUrl)

	var sumsize int64
	for _, v := range job.FileList {
		sumsize += v.CurSize
	}

	job.lastSize = sumsize

	job.Status = STATUS_LOAD
	job.download.WaitDone()

	for _, v := range job.FileList {
		if !v.Finished {
			job.Status = STATUS_STOP
			return
		}
	}

	job.Status = STATUS_DONE
	job.Finished = true
}

func TimeEqual(t1 time.Time, t2 time.Time) bool {
	now := time.Now()
	t11 := time.Date(now.Year(), now.Month(), now.Day(),
		t1.Hour(), t1.Minute(), 0,
		0, now.Location())
	t22 := time.Date(now.Year(), now.Month(), now.Day(),
		t2.Hour(), t2.Minute(), 0,
		0, now.Location())
	return t11.Equal(t22)
}

func DateEqual(t1 time.Time, t2 time.Time) bool {
	t11 := time.Date(t1.Year(), t1.Month(), t1.Day(),
		t1.Hour(), t1.Minute(), 0,
		0, t1.Location())
	t22 := time.Date(t2.Year(), t2.Month(), t2.Day(),
		t2.Hour(), t2.Minute(), 0,
		0, t2.Location())
	return t11.Equal(t22)
}

func jobReserverToQueue(cfg *KeepCfg)  {
	logs.Info("keep timeout %v", cfg)

	for _, v := range jobCtrl.cache {
		if !v.Finished && v.Reserve {
			jobToQueue(v)
			v.Reserve = false
		}
	}
}

func jobReserverTask()  {
	for {
		time.Sleep(10 * time.Second)

		jobCtrl.Lock()

		now := time.Now()
		cfg := KeepCfgGet()
		if cfg.Mode == 0 {
			if TimeEqual(now, cfg.Time) {
				jobReserverToQueue(cfg)
			}
		} else if cfg.Mode == 1 {
			if TimeEqual(now, cfg.Time) && cfg.Week[int(now.Weekday())] {
				jobReserverToQueue(cfg)
			}
		} else if cfg.Mode == 2 {
			if DateEqual(now, cfg.Time) {
				jobReserverToQueue(cfg)
			}
		}

		jobCtrl.Unlock()
	}
}

func jobSchedOnce()  {
	cfg := BaseSettingGet()

	jobCtrl.Lock()
	defer jobCtrl.Unlock()

	if cfg.MultiThreaded == len(jobCtrl.downs) {
		return
	}

	if cfg.MultiThreaded < len(jobCtrl.downs) {
		// shutdown....
		return
	}

	addnums := cfg.MultiThreaded - len(jobCtrl.downs)
	for i := 0 ; i < addnums; i++ {
		if len(jobCtrl.queue) == 0 {
			break
		}
		addJob := <- jobCtrl.queue
		if addJob.Status == STATUS_DEL {
			continue
		}
		jobCtrl.downs = append(jobCtrl.downs, addJob)
		go jobRunning(addJob)
	}
}

func jobSchedTask() {
	for  {
		time.Sleep(time.Second)
		jobSchedOnce()
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

func jobToQueue(job *Job)  {
	logs.Info("add %s job to queue", job.Timestamp)
	jobCtrl.queue <- job
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
			jobToQueue(v)
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

	go jobReserverTask()
	go jobSchedTask()
	go jobConsoleShow()

	return nil
}

