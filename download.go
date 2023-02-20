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
	video  *youtube.Video
	format *youtube.Format
	client *youtube.Client

	body       io.ReadCloser
	filestatus *DownLoadFile

	recvflow int64
	cancel   bool
	finish   chan struct{}

	file *os.File
}

const SLICE_SIZE = 64 * 1024

func (d *DownLoadTask) downloadTask() {
	var cache [SLICE_SIZE]byte

	for ;; {
		cnt, err := d.body.Read(cache[:])
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
			break;
		}
	}

	if d.filestatus.CurSize == d.filestatus.TotalSize {
		d.filestatus.Finished = true
	}

	d.body.Close()
	d.file.Close()

	d.finish <- struct{}{}

	logs.Info("download task close")
}

func DownLoadFileInit(file *DownLoadFile) (*os.File, error) {
	info, err := os.Stat(file.Filepath)
	if err != nil {
		if os.IsNotExist(err) {
			fd, err := os.Create(file.Filepath)
			if err != nil {
				logs.Error(err.Error())
				return nil, err
			}
			file.CurSize = 0
			file.Finished = false
			return fd, nil
		}
		logs.Error(err.Error())
		return nil, err
	}

	// 如果文件小于当前进度，则认为上次存储问题异常；重新下载完整内容；
	if info.Size() < file.CurSize {
		file.CurSize = 0
	}

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
	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Minute)
	body, length, err := client.GetStreamContext(ctx, video, format)
	cancelFunc()
	if err != nil {
		logs.Error(err.Error())
		return nil, err
	}

	if length != file.TotalSize {
		err = os.Remove(file.Filepath)
		if err != nil {
			logs.Error(err.Error())
			return nil, err
		}
		file.TotalSize = length
	}

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
	dl.body = body

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
