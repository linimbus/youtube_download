package main

import (
	"context"
	"fmt"
	"github.com/astaxie/beego/logs"
	"github.com/lixiangyun/youtube_download/youtube"
	"gopkg.in/yaml.v3"
	"os"
	"strings"
	"sync"
	"time"
)

type VideoDownload struct {
	sync.WaitGroup

	client    *youtube.Client
	video     *youtube.Video
	formats  []youtube.Format
	download  *DownLoad

	savesize  int64
	lastsize  int64

	shutdown  bool
	outputDir string
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

func NewVideoDownload(video *youtube.Video, weburl string, outputDir string, itagNos []int) (*VideoDownload, error) {
	err := TouchDir(outputDir)
	if err != nil {
		logs.Error(err.Error())
		return nil, err
	}

	httpclient, err := HttpClientGet(HttpProxyGet())
	if err != nil {
		logs.Error(err.Error())
		return nil, err
	}

	vdl := new(VideoDownload)
	vdl.client = &youtube.Client{
		HTTPClient: httpclient.cli,
	}

	if video == nil {
		video, err = WebVideoGet(vdl.client, weburl)
		if err != nil {
			logs.Error(err.Error())
			return nil, err
		}
	}

	vdl.outputDir = outputDir
	vdl.video = video
	vdl.formats = make([]youtube.Format, 0)

	for _, itag := range itagNos {
		for _,format := range video.Formats {
			if format.ItagNo == itag {
				vdl.formats = append(vdl.formats, format)
				break
			}
		}
	}

	videoInfomationSave(vdl)

	vdl.Add(1)
	go vdl.downLoader()

	return vdl, nil
}

type VideoInfo struct {
	Title    string
	Author   string
	Duration time.Duration
}

func videoInfomationSave(vdl *VideoDownload)  {
	value, err := yaml.Marshal(vdl.video)
	if err != nil {
		logs.Error(err.Error())
		return
	}
	filepath := fmt.Sprintf("%s\\info.txt", vdl.outputDir)
	err = SaveToFile(filepath, value)
	if err != nil {
		logs.Error(err.Error())
		return
	}
}

func FormatToFileName(f *youtube.Format) string {
	values := strings.Split(f.MimeType, ";")
	values = strings.Split(values[0],"/")

	var suffix = "mp4"
	if len(values) == 2 {
		suffix = values[1]
	}
	formatType := values[0]

	if strings.ToLower(formatType) == "audio" {
		return fmt.Sprintf("audio.m4a")
	} else {
		return fmt.Sprintf("%s[%dp].%s", formatType, f.Height, suffix)
	}
}

func (vdl *VideoDownload)downLoader() {
	defer vdl.Done()

	logs.Info("video download %s starting", vdl.outputDir)
	for _, format := range vdl.formats {
		if vdl.shutdown {
			logs.Info("download %s task shutdown", vdl.outputDir)
			break
		}

		filename := FormatToFileName(&format)
		filepath := fmt.Sprintf("%s\\%s", vdl.outputDir, filename)

		for i := 0; i < 5; i++ {
			file, err := os.Create(filepath)
			if err != nil {
				logs.Error(err.Error())
				continue
			}

			dl, err := NewDownLoad(vdl.client, vdl.video, &format, file)
			if err != nil {
				logs.Error(err.Error())
				continue
			}

			vdl.download = dl

			err = dl.WaitDone()
			if err != nil {
				logs.Error(err.Error())
				continue
			} else {
				break
			}
		}
	}

	logs.Info("video download %s finish", vdl.outputDir)
	vdl.shutdown = true
}

func (vdl *VideoDownload)Video() *youtube.Video {
	return vdl.video
}

func (vdl *VideoDownload)Formats() []youtube.Format {
	return vdl.formats
}

func (vdl *VideoDownload)Stat() int64 {
	dl := vdl.download
	if dl == nil {
		return 0
	}
	savesize, _ := dl.Stat()
	if savesize < vdl.lastsize {
		vdl.lastsize = savesize
		return savesize
	}
	tempsize := savesize - vdl.lastsize
	vdl.lastsize = savesize
	return tempsize
}

func (vdl *VideoDownload)Stop() {
	if vdl.shutdown {
		return
	}
	vdl.shutdown = true
	dl := vdl.download
	if dl != nil {
		err := dl.Cancel()
		if err != nil {
			logs.Error(err.Error())
		}
	}
}
