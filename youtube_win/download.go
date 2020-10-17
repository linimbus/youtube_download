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
	sync.Mutex

	video  * youtube.Video
	format * youtube.Format
	client * youtube.Client

	slicecache []*DownLoadSlice
	filestatus   *DownLoadFile

	cancel    bool
	finish    chan struct{}

	file     *os.File
	close     bool
}

const SLICE_SIZE = 64*1024

func (d *DownLoadMulti)downloadSlice(wg *sync.WaitGroup, rsp *DownLoadSlice, offset int64, size int64, timeout int) {
	defer wg.Done()

	for {
		if d.close {
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout) * time.Second)
		body, err := d.client.GetSliceStreamContext(ctx, d.video, d.format, offset, size)
		cancel()
		if err != nil {
			logs.Error(err.Error())
			continue
		}

		//fmt.Printf("offset: %d, size: %d, body: %d\n", offset, size, len(body))

		rsp.body = body
		return
	}
}

func (d *DownLoadMulti)syncfile() {

retry:
	for _, v := range d.slicecache {
		if v.offset != d.filestatus.CurSize {
			continue
		}

		if v.body == nil || v.size == 0 {
			continue
		}

		d.file.Seek(v.offset, 0)
		err := WriteFull(d.file, v.body)
		if err != nil {
			logs.Error(err.Error())
		}

		d.file.Sync()
		d.filestatus.CurSize += v.size

		v.body = nil
		v.size = 0
		v.offset = 0

		goto retry
	}
}

func (d *DownLoadMulti)sliceNoblock() *DownLoadSlice {
	for _, v := range d.slicecache {
		if v.size == 0 {
			return v
		}
	}
	return nil
}

func (d *DownLoadMulti)downloadTask() {
	wg := new(sync.WaitGroup)

	curSize := d.filestatus.CurSize
	totalSize := d.filestatus.TotalSize

	for offset := curSize; offset < totalSize; {
		slicesize := int64(SLICE_SIZE)
		if offset + slicesize > totalSize {
			slicesize = totalSize - offset
		}
		for  {
			if d.cancel {
				goto shutdown
			}
			node := d.sliceNoblock()
			if node == nil {
				d.syncfile()
				time.Sleep(50 * time.Millisecond)
				continue
			}
			node.offset = offset
			node.size = slicesize
			node.body = nil
			wg.Add(1)
			go d.downloadSlice(wg, node, offset, slicesize, 10)
			break
		}
		offset += slicesize
	}

shutdown:

	wg.Wait()
	d.syncfile()

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
	dl.format = format
	dl.client = client
	dl.video = video
	dl.filestatus = file
	dl.file = fd
	dl.slicecache = make([]*DownLoadSlice, 20)
	for i, _ := range dl.slicecache {
		dl.slicecache[i] = &DownLoadSlice{}
	}

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

func (d *DownLoadMulti)Stat() (int64, int64){
	return d.filestatus.CurSize, d.filestatus.TotalSize
}
