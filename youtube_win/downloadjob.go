package main

import (
	"context"
	"errors"
	"github.com/astaxie/beego/logs"
	"github.com/lixiangyun/youtube_download/youtube"
	"time"
)

type DownloadJob struct {
	close      chan struct{}
	cancel     chan struct{}
	cancelFlag bool

	weburl     string
	jobID      string

	dlmulti   *DownLoadMulti
	filelist  []DownLoadFile
}

func WebVideoGet(client *youtube.Client, weburl string) (*youtube.Video, error) {
	var err error
	var video *youtube.Video

	for i := 0; i < 5; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		video, err = client.GetVideoContext(ctx, weburl)
		cancel()
		if err != nil {
			logs.Error(err.Error())
			continue
		}
		if video != nil {
			break
		}
	}

	if err != nil {
		logs.Error(err.Error())
		return nil, err
	}

	return video, nil
}

var ErrNoExitItagNo = errors.New("video info itagno not exist!")

func WebDownloadClient() (*youtube.Client, error ) {
	httpclient, err := HttpClientGet(HttpProxyGet())
	if err != nil {
		logs.Error(err.Error())
		return nil, err
	}
	return &youtube.Client{HTTPClient: httpclient.cli}, nil
}

func WebVideoInfoGet(client *youtube.Client, weburl string, file *DownLoadFile) (*youtube.Video, *youtube.Format, error) {
	video, err := WebVideoGet(client, weburl)
	if err != nil {
		logs.Error(err.Error())
		return nil, nil, err
	}

	for _, format := range video.Formats {
		if format.ItagNo == file.ItagNo {
			return video, &format, nil
		}
	}

	return nil, nil, ErrNoExitItagNo
}

func NewDownloadJob(jobID string, weburl string, fileList []DownLoadFile) (*DownloadJob, error) {
	vdl := new(DownloadJob)
	vdl.jobID = jobID
	vdl.weburl = weburl
	vdl.filelist = fileList
	vdl.cancel = make(chan struct{}, 10)
	vdl.close = make(chan struct{}, 10)

	go vdl.downLoaderJob()
	return vdl, nil
}

func (vdl *DownloadJob)downLoadFileGet() *DownLoadFile {
	for i, _ := range vdl.filelist {
		file := &vdl.filelist[i]
		if file.Finished {
			continue
		}
		return file
	}
	return nil
}

func (vdl *DownloadJob)downLoaderJob() {
	logs.Info("download job %s start", vdl.jobID)

	for  {
		if vdl.cancelFlag {
			logs.Info("download job cancel done")
			break
		}

		file := vdl.downLoadFileGet()
		if file == nil {
			logs.Info("download job all file done")
			break
		}

		client, err := WebDownloadClient()
		if err != nil {
			logs.Error("download download client fail, %s", err.Error())
			break
		}

		var video  * youtube.Video
		var format * youtube.Format

		for i := 0 ; i < 5; i++ {
			video, format, err = WebVideoInfoGet(client, vdl.weburl, file)
			if err != nil {
				logs.Error(err.Error())
				time.Sleep(time.Second * 2)
				continue
			}
			break
		}

		if err != nil {
			logs.Info("download job exception close")
			break
		}

		for i := 0 ; i < 5; i++ {
			vdl.dlmulti, err = NewDownLoadMulti(client, video, format, file)
			if err != nil {
				logs.Error(err.Error())
				time.Sleep(time.Second * 2)
				continue
			}
			break
		}

		if err != nil {
			logs.Info("download job exception close")
			break
		}

		if vdl.dlmulti != nil {
			select {
				case <- vdl.dlmulti.Finish(): {
					continue
				}
				case <- vdl.cancel: {
					vdl.dlmulti.Cancel()
					break
				}
			}
		}

		time.Sleep(time.Second * 5)
	}

	logs.Info("download job %s close", vdl.jobID)
	vdl.close <- struct {}{}
}

func (vdl *DownloadJob)WaitDone() {
	<- vdl.close
	logs.Warn("download job %s done", vdl.jobID)
}

func (vdl *DownloadJob)Cancel() {
	vdl.cancel <- struct{}{}
	vdl.cancelFlag = true
	logs.Warn("download job %s cancel", vdl.jobID)
}

func (vdl *DownloadJob)Flow() int64 {
	dl := vdl.dlmulti
	if dl != nil {
		return dl.Flow()
	}
	return 0
}