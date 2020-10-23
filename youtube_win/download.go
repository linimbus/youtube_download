package main

import (
	"context"
	"github.com/astaxie/beego/logs"
	"github.com/lixiangyun/youtube_download/youtube"
	"os"
	"sync"
	"sync/atomic"
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

type DownLoadReq struct {
	offset int64
	size   int64
}

type DownLoadMulti struct {
	video  * youtube.Video
	format * youtube.Format
	client * youtube.Client

	slicereq   chan *DownLoadReq
	slicecache chan *DownLoadSlice

	filestatus   *DownLoadFile

	recvflow  int64
	timeout   int
	cancel    bool
	finish    chan struct{}

	file     *os.File
}

const SLICE_SIZE = 64*1024

var dlSpeedLimitDelay time.Duration

func DownLoadSpeedLimitDelayAdd()  {
	dlSpeedLimitDelay++
	logs.Info("download speed limit delay : %d s", dlSpeedLimitDelay )
}

func DownLoadSpeedLimitDelayDel()  {
	if dlSpeedLimitDelay > 0 {
		dlSpeedLimitDelay--
	} else {
		dlSpeedLimitDelay = 0
	}
	logs.Info("download speed limit delay : %d s", dlSpeedLimitDelay )
}

func DownLoadSpeedLimitDisable()  {
	dlSpeedLimitDelay = 0
}

func downLoadSpeedLimit()  {
	if dlSpeedLimitDelay > 0 {
		time.Sleep(dlSpeedLimitDelay * time.Second)
	}
}

func (d *DownLoadMulti)downloadSlice(wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		req, exist := <- d.slicereq
		if exist == false {
			logs.Info("download slice task exit")
			return
		}

		var body []byte
		var err error

		for {
			downLoadSpeedLimit()

			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(d.timeout) * time.Second)
			body, err = d.client.GetSliceStreamContext(ctx, d.video, d.format, req.offset, req.size)
			cancel()
			if err == nil {
				break
			}
			logs.Error(err.Error())
			time.Sleep(2 * time.Second)

			if d.cancel {
				logs.Info("download slice task cancel")
				for  {
					_,exist := <- d.slicereq
					if exist == false {
						break
					}
				}
				return
			}
		}

		d.slicecache <- &DownLoadSlice{body: body, offset: req.offset, size: req.size}
	}
}

func (d *DownLoadMulti)downloadRecv(wg *sync.WaitGroup)  {
	defer wg.Done()

	var sliceList []*DownLoadSlice

	for  {
		slice, exist := <- d.slicecache
		if exist == false {
			logs.Info("download slice recv exit")
			return
		}

		atomic.AddInt64(&d.recvflow, slice.size)
		sliceList = append(sliceList, slice)

		sliceList = syncfile(sliceList, d.filestatus, d.file)
	}
}

func syncfile(cache []*DownLoadSlice, file *DownLoadFile, fd *os.File) []*DownLoadSlice {
retry:

	for i, v := range cache {
		if v.offset == file.CurSize {
			cache = append(cache[:i], cache[i+1:]...)

			fd.Seek(v.offset, 0)
			err := WriteFull(fd, v.body)
			if err != nil {
				logs.Error(err.Error())
			}
			fd.Sync()
			file.CurSize += v.size

			goto retry
		}
	}

	return cache
}

func (d *DownLoadMulti)downloadTask() {
	wg := new(sync.WaitGroup)
	wg2 := new(sync.WaitGroup)

	step := 10
	d.timeout = step * 6

	curSize := d.filestatus.CurSize
	totalSize := d.filestatus.TotalSize

	for i:=0; i < 5; i++ {
		wg.Add(1)
		go d.downloadSlice(wg)
	}

	wg2.Add(1)
	go d.downloadRecv(wg2)

	for offset := curSize; offset < totalSize; {
		slicesize := int64(SLICE_SIZE * step)
		if offset + slicesize > totalSize {
			slicesize = totalSize - offset
		}

		d.slicereq <- &DownLoadReq{offset: offset, size: slicesize}
		if d.cancel {
			logs.Info("download task cancel")
			goto shutdown
		}

		offset += slicesize
	}

shutdown:
	close(d.slicereq)
	wg.Wait()

	close(d.slicecache)
	wg2.Wait()

	if d.filestatus.CurSize == d.filestatus.TotalSize {
		d.filestatus.Finished = true
	}

	d.finish <- struct{}{}
	d.file.Close()

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
	dl.format = format
	dl.client = client
	dl.video = video
	dl.filestatus = file
	dl.file = fd
	dl.slicereq   = make(chan *DownLoadReq, 10)
	dl.slicecache = make(chan *DownLoadSlice, 10)

	dl.finish = make(chan struct{}, 10)

	go dl.downloadTask()

	return dl, nil
}

func (d *DownLoadMulti)Finish() chan struct{}{
	return d.finish
}

func (d *DownLoadMulti)Cancel() {
	d.cancel = true
}

func (d *DownLoadMulti)Flow() (int64){
	return d.recvflow
}

