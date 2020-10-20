package main

import (
	"context"
	"github.com/astaxie/beego/logs"
	"github.com/lixiangyun/youtube_download/youtube"
	"time"
)

type DownloadJob struct {
	close      chan struct{}
	cancel     chan struct{}
	cancelFlag bool

	jobID      string
	client    *youtube.Client
	video     *youtube.Video
	formats   []youtube.Format
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

func NewDownloadJob(jobID string, weburl string, fileList []DownLoadFile) (*DownloadJob, error) {
	httpclient, err := HttpClientGet(HttpProxyGet())
	if err != nil {
		logs.Error(err.Error())
		return nil, err
	}

	vdl := new(DownloadJob)
	vdl.client = &youtube.Client{
		HTTPClient: httpclient.cli,
	}

	video, err := WebVideoGet(vdl.client, weburl)
	if err != nil {
		logs.Error(err.Error())
		return nil, err
	}

	vdl.video = video
	vdl.formats = make([]youtube.Format, 0)

	for _, fileinfo := range fileList {
		for _,format := range video.Formats {
			if format.ItagNo == fileinfo.ItagNo {
				vdl.formats = append(vdl.formats, format)
				break
			}
		}
	}

	vdl.jobID = jobID
	vdl.filelist = fileList
	vdl.cancel = make(chan struct{}, 10)
	vdl.close = make(chan struct{}, 10)

	go vdl.downLoaderJob()

	return vdl, nil
}

func (vdl *DownloadJob)downLoaderJob() {
	logs.Info("download job %s start", vdl.jobID)

	for _, format := range vdl.formats {
		if vdl.cancelFlag {
			break
		}

		var fileInfo *DownLoadFile
		for i, v := range vdl.filelist {
			if v.ItagNo == format.ItagNo {
				fileInfo = &vdl.filelist[i]
			}
		}

		if fileInfo.Finished {
			continue
		}

		var err error
		for i := 0 ; i < 5; i++ {
			vdl.dlmulti, err = NewDownLoadMulti(vdl.client, vdl.video, &format, fileInfo)
			if err != nil {
				logs.Error(err.Error())
				continue
			}
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