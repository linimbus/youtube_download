package main

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/astaxie/beego/logs"
	"github.com/kkdai/youtube/v2"
)

type DownLoadFile struct {
	ItagNo    int
	Filepath  string
	CurSize   int64
	TotalSize int64
	Finished  bool
}

type DownLoadTask struct {
	video      *youtube.Video
	format     *youtube.Format
	client     *youtube.Client
	filestatus *DownLoadFile

	recvflow int64
	cancel   bool
	finish   chan struct{}

	file *os.File
}

const SLICE_SIZE = 64 * 1024

func (d *DownLoadTask) downloadTask() {
	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Minute)
	body, length, err := d.client.GetStreamContext(ctx, d.video, d.format)

	if err != nil {
		logs.Error(err.Error())
	} else {
		var cache [SLICE_SIZE]byte
		d.filestatus.TotalSize = length
		for {
			cnt, err := body.Read(cache[:])
			if cnt > 0 {
				err = WriteFull(d.file, cache[:cnt])
				if err != nil {
					logs.Error(err.Error())
				}
				d.file.Sync()

				d.recvflow += int64(cnt)
				d.filestatus.CurSize += int64(cnt)
			}
			if err != nil {
				if err != io.EOF {
					logs.Warning("download task read io fail, %s", err.Error())
				}
				break
			}
		}
		body.Close()
	}

	if d.filestatus.CurSize == d.filestatus.TotalSize {
		d.filestatus.Finished = true
	}

	d.file.Close()
	cancelFunc()

	d.finish <- struct{}{}

	logs.Info("download task close")
}

func DownLoadFileInit(file *DownLoadFile) (*os.File, error) {
	os.Remove(file.Filepath)
	fd, err := os.OpenFile(file.Filepath, os.O_RDWR, 0644)
	if err != nil {
		logs.Error(err.Error())
		return nil, err
	}
	return fd, nil
}

func NewDownLoad(client *youtube.Client,
	video *youtube.Video,
	format *youtube.Format,
	file *DownLoadFile) (*DownLoadTask, error) {

	fd, err := DownLoadFileInit(file)
	if err != nil {
		if err != nil {
			logs.Error(err.Error())
			return nil, err
		}
	}

	dl := new(DownLoadTask)
	dl.format = format
	dl.client = client
	dl.video = video
	dl.filestatus = file
	dl.file = fd

	dl.finish = make(chan struct{}, 10)

	go dl.downloadTask()

	return dl, nil
}

func (d *DownLoadTask) Finish() chan struct{} {
	return d.finish
}

func (d *DownLoadTask) Cancel() {
	d.cancel = true
}

func (d *DownLoadTask) Flow() int64 {
	return d.recvflow
}
