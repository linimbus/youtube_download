package main

import (
	"context"
	"github.com/astaxie/beego/logs"
	"github.com/lixiangyun/youtube_download/youtube"
	"os"
	"sync"
	"time"
)

type DownLoadFile struct {
	ItagNo    int
	Filepath  string
	CurSize   int64
	TotalSize int64
	Finished  bool
}

type DownLoadSlice struct {
	body   []byte
	offset int64
	size   int64
}

type DownLoadMulti struct {
	video  * youtube.Video
	format * youtube.Format
	client * youtube.Client

	sliceRecv  chan *DownLoadSlice
	filestatus *DownLoadFile

	cancel      chan struct{}
	finish      chan struct{}

	file     *os.File
	close     bool
}

const SLICE_SIZE = 128*1024

func (d *DownLoadMulti)downloadSlice(wg *sync.WaitGroup, sem *SemPV, offset int64, size int64) {
	defer wg.Done()
	defer sem.SemV()

	for {
		if d.close {
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		body, err := d.client.GetSliceStreamContext(ctx, d.video, d.format, offset, size)
		cancel()
		if err != nil {
			logs.Error(err.Error())
			continue
		}

		d.sliceRecv <- &DownLoadSlice{body: body, size: size, offset: offset}
		return
	}
}

func (d *DownLoadMulti)syncfile(cache []*DownLoadSlice) {
retry:
	for i, v := range cache {
		if v.offset != d.filestatus.CurSize {
			continue
		}
		cache = append(cache[:i], cache[i+1:]...)

		d.file.Seek(v.offset, 0)
		err := WriteFull(d.file, v.body)
		if err != nil {
			logs.Error(err.Error())
		}
		d.file.Sync()
		d.filestatus.CurSize += v.size

		goto retry
	}
}

func (d *DownLoadMulti)downloadRecv(wg *sync.WaitGroup, stop chan struct{})  {
	defer wg.Done()

	var cache []*DownLoadSlice

	for  {
		select {
		case slice := <- d.sliceRecv: {
			cache = append(cache, slice)
			d.syncfile(cache)
		}
		case <- stop:
			break
		}
	}

	cnt := len(d.sliceRecv)

	for i := 0; i < cnt; i++ {
		slice := <- d.sliceRecv
		cache = append(cache, slice)
		d.syncfile(cache)
	}
}

func (d *DownLoadMulti)downloadTask() {
	stop := make(chan struct{}, 1)
	wg := new(sync.WaitGroup)

	go d.downloadRecv(wg, stop)

	sem := SemInit(10)

	length := d.filestatus.TotalSize - d.filestatus.CurSize
	sliceCnt := length / SLICE_SIZE
	sliceEnd := length % SLICE_SIZE

	for i := 0 ; i < int(sliceCnt) ; i++  {
		offset := int64(i*SLICE_SIZE) + d.filestatus.CurSize
		select {
			case <- sem.queue: {
				wg.Add(1)
				go d.downloadSlice(wg, sem, offset, SLICE_SIZE)
			}
			case <- d.cancel: {
				goto shutdown
			}
		}
	}
	wg.Add(1)
	go d.downloadSlice(wg, sem, sliceCnt*SLICE_SIZE, sliceEnd)

shutdown:
	d.close = true
	stop <- struct{}{}

	wg.Wait()

	if d.filestatus.CurSize == d.filestatus.TotalSize {
		d.filestatus.Finished = true
	}

	logs.Info("download task close")

	d.finish <- struct{}{}
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

	fd, err := os.Open(file.Filepath)
	if err != nil {
		logs.Error(err.Error())
		return nil, err
	}

	return fd, nil
}

func NewDownLoadMulti(client * youtube.Client,
					  video * youtube.Video,
					  format * youtube.Format,
					  file *DownLoadFile ) (*DownLoadMulti, error) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Minute)
	length, err := client.GetStreamContextLangth(ctx, video, format)
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

	dl := new(DownLoadMulti)
	dl.cancel = make(chan struct{}, 10)
	dl.format = format
	dl.client = client
	dl.video = video
	dl.filestatus = file
	dl.file = fd
	dl.sliceRecv = make(chan *DownLoadSlice, 20)
	dl.finish = make(chan struct{}, 10)

	go dl.downloadTask()

	return dl, nil
}

func (d *DownLoadMulti)Finish() chan struct{}{
	return d.finish
}

func (d *DownLoadMulti)Cancel() {
	d.cancel <- struct{}{}
}

func (d *DownLoadMulti)Stat() (int64, int64){
	return d.filestatus.CurSize, d.filestatus.TotalSize
}
