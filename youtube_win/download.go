package main

import (
	"context"
	"github.com/astaxie/beego/logs"
	"github.com/lixiangyun/youtube_download/youtube"
	"io"
	"sync"
	"sync/atomic"
)

type DownLoad struct {
	sync.WaitGroup

	cancelfunc context.CancelFunc
	close     bool
	err       error
	savesize  int64
	langth    int64
}

func (d *DownLoad)downloadCopy(dest io.WriteCloser, src io.ReadCloser)  {
	var buffer [4096]byte
	for  {
		cnt, err1 := src.Read(buffer[:])
		if cnt > 0 {
			atomic.AddInt64(&d.savesize, int64(cnt))
			err2 := WriteFull(dest, buffer[:cnt])
			if err2 != nil {
				d.err = err2
				break
			}
		}
		if err1 != nil {
			d.err = err1
			break
		}
	}

	if d.err != nil {
		logs.Error(d.err.Error())
	}

	src.Close()
	dest.Close()
	d.Done()

	d.close = true
	logs.Info("download thread close")
}

func NewDownLoad(client * youtube.Client, video * youtube.Video, format * youtube.Format, file io.WriteCloser ) (*DownLoad, error) {
	context, cancelfunc := context.WithCancel(context.Background())
	rsp, err := client.GetStreamContext(context, video, format)
	if err != nil {
		logs.Error(err.Error())
		return nil, err
	}

	dl := new(DownLoad)
	dl.cancelfunc = cancelfunc
	dl.langth = rsp.ContentLength
	dl.Add(1)
	go dl.downloadCopy(file, rsp.Body)

	return dl, nil
}

func (d *DownLoad)WaitDone()  {
	d.Wait()
	logs.Info("download wait success")
}

func (d *DownLoad)Cancel() error {
	if d.close {
		return nil
	}
	d.cancelfunc()
	d.Wait()
	return d.err
}

func (d *DownLoad)Stat() (int64, int64){
	return d.savesize, d.langth
}


