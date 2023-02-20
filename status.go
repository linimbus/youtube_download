package main

import (
	"fmt"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"time"
)

var statusFlow *walk.StatusBarItem
var statusTime *walk.StatusBarItem

func statusFlowGet(flow int) string {
	if flow == 0 {
		return " "
	}
	return fmt.Sprintf(" %s/s", ByteViewLite(int64(flow)) )
}

func UpdateStatusFlow(flow int)  {
	if statusFlow != nil {
		statusFlow.SetText(statusFlowGet(flow) )
	}
}

func TimestampUpdate()  {
	if statusTime != nil {
		statusTime.SetText(GetTimeStamp())
	}
}

func init()  {
	go func() {
		for  {
			time.Sleep(time.Second)
			TimestampUpdate()
		}
	}()
}

func StatusBarInit() []StatusBarItem {
	return []StatusBarItem{
		{
			Icon: ICON_Network_Flow,
			Width: 16,
		},
		{
			AssignTo: &statusFlow,
			ToolTipText: LangValue("realtimeflow"),
			Width: 80,
			Text: statusFlowGet(0),
		},
		{
			AssignTo: &statusTime,
			Width: 120,
			Text: GetTimeStamp(),
		},
	}
}
