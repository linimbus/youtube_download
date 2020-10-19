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
	From        string
	OutputDir   string
	FileList    []DownLoadFile

	flowSize    int64

	TotalSize   int64
	Status      string
	DeleteFile  bool

	SpeedLast []int64

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

func videoContentLangthGet(video *youtube.Video, format *youtube.Format) (int64,error) {
	httpclient, err := HttpClientGet(HttpProxyGet())
	if err != nil {
		logs.Error(err.Error())
		return 0, err
	}
	var length int64
	client := &youtube.Client{HTTPClient: httpclient.cli}
	for i :=0 ; i < 5; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 15 * time.Second)
		length, err = client.GetStreamContextLangth(ctx, video, format)
		cancel()
		if err != nil {
			logs.Error(err.Error())
			continue
		}
		return length, nil
	}
	return 0, err
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

func (job *Job)Finished() bool {
	for _, v := range job.FileList {
		if !v.Finished {
			return false
		}
	}
	return true
}

func JobAdd(video *youtube.Video, itagno []int, weburl string, reserve bool) error {
	job := new(Job)
	job.Timestamp = GetTimeStampNumber()
	job.WebUrl = weburl
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
				contentLength, err := videoContentLangthGet(video, &format)
				if err != nil {
					return err
				}
				totalSize += contentLength
				fileList = append(fileList, DownLoadFile{
					ItagNo: v,
					TotalSize: contentLength,
					Filepath: fmt.Sprintf("%s\\%s", job.OutputDir,
						videoFormatFileName(&format)),
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
	jobCtrl.Unlock()

	return nil
}

var jobCtrl *JobCtrl

func RemainCalc(speed int64, totalsize int64) time.Duration {
	if speed == 0 {
		return -1
	}
	return time.Second * time.Duration(totalsize / speed)
}

func (j * Job)SpeedAvg(speed int64) int64 {
	if len(j.SpeedLast) > 5 {
		j.SpeedLast = j.SpeedLast[1:]
	}
	j.SpeedLast = append(j.SpeedLast, speed)
	var output int64
	for _, v := range j.SpeedLast {
		output += v
	}
	return output/int64(len(j.SpeedLast))
}

func job2Item(i int, job *Job) *JobItem {
	var speed int64

	var sumsize int64
	for _, v := range job.FileList {
		sumsize += v.CurSize
	}

	var flowsize int64
	dl := job.download
	if dl != nil {
		flowsize = dl.Flow()
	}

	if job.flowSize > flowsize {
		speed = flowsize
	} else {
		speed = flowsize - job.flowSize
	}
	job.flowSize = flowsize

	speed = job.SpeedAvg(speed)
	if job.Status != STATUS_LOAD {
		speed = 0
	}

	rate := int64(100)
	if !job.Finished() {
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
		Size: job.TotalSize,
		From: job.From,
		Status: job.Status,
		outputDir: job.OutputDir,
		Remaind: RemainCalc(speed, job.TotalSize - sumsize ),
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

func JobLoading(list []string)  {
	jobCtrl.Lock()
	defer jobCtrl.Unlock()

	logs.Info("job reloading: %v", list)

	for _, v := range list {
		for _, job := range jobCtrl.cache {
			if job.Timestamp == v {
				if job.Status == STATUS_RESV || job.Status == STATUS_STOP {
					jobToQueue(job)
				}
				break
			}
		}
	}
}

func JobStop(list []string) {
	jobCtrl.Lock()
	defer jobCtrl.Unlock()

	logs.Warn("job stop: %v", list)

	for _, v := range list {
		for _, job := range jobCtrl.downs {
			if job.Timestamp == v {
				dl := job.download
				if dl != nil {
					dl.Cancel()
				}
				break
			}
		}
	}
}

func JobDelete(list []string, file bool) error {
	jobCtrl.Lock()
	defer jobCtrl.Unlock()

	logs.Warn("job delete: %v, with delete file", list)

	for _, v := range list {
		for i, job := range jobCtrl.cache {
			if job.Timestamp == v {
				dl := job.download
				if dl != nil {
					job.DeleteFile = true
					job.Status = STATUS_DEL

					dl.Cancel()
				} else {
					if file {
						RemoveAllFile(job.OutputDir)
					}
				}
				jobCtrl.cache = append(jobCtrl.cache[:i], jobCtrl.cache[i+1:]...)
				break
			}
		}
	}

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
		jobRemoveDowningList(job)
		jobCtrl.Unlock()
	}()

	var err error
	job.download, err = NewDownloadJob(job.Timestamp, job.WebUrl, job.FileList )
	if err != nil {
		logs.Error(err.Error())
		job.Status = STATUS_STOP
		return
	}

	logs.Info("video download task running: %s", job.WebUrl)

	job.Status = STATUS_LOAD
	job.download.WaitDone()

	if job.Status == STATUS_DEL {
		if job.DeleteFile {
			RemoveAllFile(job.OutputDir)
		}
		logs.Info("video download task delete: %s", job.WebUrl)
		return
	}

	for _, v := range job.FileList {
		if !v.Finished {
			job.Status = STATUS_STOP
			logs.Info("video download task stop: %s", job.WebUrl)
			return
		}
	}

	job.Status = STATUS_DONE
	logs.Info("video download task done: %s", job.WebUrl)
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
	logs.Info("job reserver to queue %v", cfg)
	for _, v := range jobCtrl.cache {
		if v.Status == STATUS_RESV {
			jobToQueue(v)
		}
	}
}

func jobReserverTask()  {
	for {
		time.Sleep(15 * time.Second)

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

func jobToDowningList(job *Job)  {
	jobCtrl.downs = append(jobCtrl.downs, job)
}

func jobRemoveDowningList(job *Job)  {
	for i, v := range jobCtrl.downs {
		if v == job {
			jobCtrl.downs = append(jobCtrl.downs[:i], jobCtrl.downs[i+1:]...)
			return
		}
	}
}

func jobLastDowningList(num int) []string {
	length := len(jobCtrl.downs)
	var output []string
	for _,v := range jobCtrl.downs[length-num:] {
		output = append(output, v.Timestamp)
	}
	return output
}

func jobSchedOnce()  {
	cfg := BaseSettingGet()

	jobCtrl.Lock()
	defer jobCtrl.Unlock()

	if cfg.MultiThreaded == len(jobCtrl.downs) {
		return
	}

	if cfg.MultiThreaded < len(jobCtrl.downs) {
		list := jobLastDowningList( len(jobCtrl.downs) - cfg.MultiThreaded)
		go JobStop(list)
		return
	}

	addnums := cfg.MultiThreaded - len(jobCtrl.downs)
	for i := 0 ; i < addnums; i++ {
		if len(jobCtrl.queue) == 0 {
			break
		}
		addJob := <- jobCtrl.queue
		if addJob.Status != STATUS_WAIT {
			continue
		}
		jobToDowningList(addJob)
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
	job.Status = STATUS_WAIT
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
		logs.Error("parse job config fail, %s", err.Error())
		return nil
	}

	for _, v := range output {
		temp := v
		jobCtrl.cache = append(jobCtrl.cache, &temp)
	}

	for _, v := range jobCtrl.cache {
		if !v.Finished() && v.Status != STATUS_RESV {
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

