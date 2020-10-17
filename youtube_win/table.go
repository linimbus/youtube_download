package main

import (
	"fmt"
	"github.com/astaxie/beego/logs"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"sort"
	"sync"
)

type JobItem struct {
	Index        int
	Title        string
	ProgressRate int
	Speed        int
	Size         int64
	From         string
	Status       string

	outputDir    string
	checked      bool
}

type JobModel struct {
	sync.RWMutex

	walk.TableModelBase
	walk.SorterBase
	sortColumn int
	sortOrder  walk.SortOrder

	items      []*JobItem
}

func (n *JobModel)RowCount() int {
	return len(n.items)
}

func (n *JobModel)Value(row, col int) interface{} {
	item := n.items[row]
	switch col {
	case 0:
		return item.Index
	case 1:
		return item.Title
	case 2:
		return fmt.Sprintf("%d%%", item.ProgressRate)
	case 3:
		if item.Speed == 0 {
			return "-"
		}
		return fmt.Sprintf("%s/s", ByteViewLite(int64(item.Speed)))
	case 4:
		return ByteView(item.Size)
	case 5:
		return item.From
	case 6:
		return item.Status
	}
	panic("unexpected col")
}

func (n *JobModel) Checked(row int) bool {
	return n.items[row].checked
}

func (n *JobModel) SetChecked(row int, checked bool) error {
	n.items[row].checked = checked
	return nil
}

func (m *JobModel) Sort(col int, order walk.SortOrder) error {
	m.sortColumn, m.sortOrder = col, order
	sort.SliceStable(m.items, func(i, j int) bool {
		a, b := m.items[i], m.items[j]
		c := func(ls bool) bool {
			if m.sortOrder == walk.SortAscending {
				return ls
			}
			return !ls
		}
		switch m.sortColumn {
		case 0:
			return c(a.Index < b.Index)
		case 1:
			return c(a.Title < b.Title)
		case 2:
			return c(a.ProgressRate < b.ProgressRate)
		case 3:
			return c(a.Speed < b.Speed)
		case 4:
			return c(a.Size < b.Size)
		case 5:
			return c(a.From < b.From)
		case 6:
			return c(a.Status < b.Status)
		}
		panic("unreachable")
	})
	return m.SorterBase.Sort(col, order)
}

const (
	STATUS_STOP = "stop"
	STATUS_DONE = "done"
	STATUS_WAIT = "wait"
	STATUS_LOAD = "loading"
	STATUS_RESV = "reserver"
	STATUS_DEL  = "delete"
)

func StatusToIcon(status string) walk.Image {
	switch status {
	case STATUS_STOP:
		return ICON_STATUS_STOP
	case STATUS_DONE:
		return ICON_STATUS_DONE
	case STATUS_WAIT:
		return ICON_STATUS_WAIT
	case STATUS_RESV:
		return ICON_STATUS_RESERVER
	case STATUS_LOAD:
		return ICON_STATUS_LOAD
	default:
		return ICON_STATUS_WAIT
	}
	return nil
}

var jobBitmap *walk.Bitmap

var jobTable *JobModel

func init()  {
	jobTable = new(JobModel)
	jobTable.items = make([]*JobItem, 0)
}

func JobTalbeUpdate(item []*JobItem )  {
	jobTable.Lock()
	defer jobTable.Unlock()

	oldItem := jobTable.items
	if len(oldItem) == len(item) {
		for i, v := range item {
			v.checked = oldItem[i].checked
		}
	}

	jobTable.items = item
	jobTable.PublishRowsReset()
	jobTable.Sort(jobTable.sortColumn, jobTable.sortOrder)
}

func JobDir(idx int) string {
	jobTable.RLock()
	defer jobTable.RUnlock()

	if idx >= 0 && idx < len(jobTable.items) {
		return jobTable.items[idx].outputDir
	}

	return ""
}

func JobTableSelectAll()  {
	jobTable.Lock()
	defer jobTable.Unlock()

	done := true
	for _, v := range jobTable.items {
		if !v.checked {
			done = false
		}
	}

	for _, v := range jobTable.items {
		v.checked = !done
	}

	jobTable.PublishRowsReset()
	jobTable.Sort(jobTable.sortColumn, jobTable.sortOrder)
}

func JobTableSelectList() []string {
	jobTable.RLock()
	defer jobTable.RUnlock()

	var output []string
	for _, v := range jobTable.items {
		if v.checked {
			output = append(output, v.Title)
		}
	}

	return output
}

func JobTableSelectStatus(status string)  {
	jobTable.Lock()
	defer jobTable.Unlock()

	for _, v := range jobTable.items {
		v.checked = false
	}

	for _, v := range jobTable.items {
		if v.Status == status {
			v.checked = true
		}
	}

	jobTable.PublishRowsReset()
	jobTable.Sort(jobTable.sortColumn, jobTable.sortOrder)
}

var tableView *walk.TableView

func TableWight() []Widget {
	var err error

	jobBitmap, err = walk.NewBitmap(walk.Size{100, 1})
	if err != nil {
		logs.Error("new bit map fail, %s", err.Error())
	} else {
		canvas, err := walk.NewCanvasFromImage(jobBitmap)
		if err != nil {
			logs.Error(err.Error())
		} else {
			canvas.GradientFillRectangle(
				walk.RGB(0, 205, 0),
				walk.RGB(0, 205, 0),
				walk.Horizontal,
				walk.Rectangle{0, 0, 100, 1})
			canvas.Dispose()
		}
	}

	return []Widget{
		Label{
			Text: LangValue("downloadlist"),
		},
		TableView{
			AssignTo: &tableView,
			AlternatingRowBG: true,
			ColumnsOrderable: true,
			CheckBoxes: true,
			OnItemActivated: func() {
				dir := JobDir(tableView.CurrentIndex())
				if dir != "" {
					OpenBrowserWeb(dir)
				}
			},
			Columns: []TableViewColumn{
				{Title: "#", Width: 30},
				{Title: LangValue("title"), Width: 160},
				{Title: LangValue("progressrate"), Width: 120},
				{Title: LangValue("speed"), Width: 80},
				{Title: LangValue("size"), Width: 80},
				{Title: LangValue("from"), Width: 120},
				{Title: LangValue("status"), Width: 80},
			},
			StyleCell: func(style *walk.CellStyle) {
				item := jobTable.items[style.Row()]

				if style.Row()%2 == 0 {
					style.BackgroundColor = walk.RGB(248, 248, 255)
				} else {
					style.BackgroundColor = walk.RGB(220, 220, 220)
				}

				switch style.Col() {
				case 2:
					if canvas := style.Canvas(); canvas != nil {
						bounds := style.Bounds()
						bounds2 := bounds

						bounds.Width = int(float64(bounds.Width) * float64(item.ProgressRate))/100
						bounds.Height -= 1
						canvas.DrawBitmapPartWithOpacity(jobBitmap,
							bounds,
							walk.Rectangle{0, 0, item.ProgressRate, 1},
							80)

						canvas.DrawText(fmt.Sprintf("%d%%", item.ProgressRate), tableView.Font(), 0, bounds2, walk.TextLeft)
					}
				case 6:
					style.Image = StatusToIcon(item.Status)
				}
			},
			Model:jobTable,
		},
		Composite{
			Layout: HBox{MarginsZero: true},
			Children: []Widget{
				PushButton{
					Text: LangValue("all"),
					OnClicked: func() {
						go func() {
							JobTableSelectAll()
						}()
					},
				},
				PushButton{
					Text: LangValue("statusdone"),
					OnClicked: func() {
						go func() {
							JobTableSelectStatus(STATUS_DONE)
						}()
					},
				},
				PushButton{
					Text: LangValue("statusstop"),
					OnClicked: func() {
						go func() {
							JobTableSelectStatus(STATUS_STOP)
						}()
					},
				},
				PushButton{
					Text: LangValue("statusload"),
					OnClicked: func() {
						go func() {
							JobTableSelectStatus(STATUS_LOAD)
						}()
					},
				},
				HSpacer{

				},
			},
		},
	}
}

