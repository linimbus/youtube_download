package main

import (
	"fmt"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"math/rand"
	"sort"
	"sync"
)

type JobItem struct {
	Index        int
	Title        string
	ProgressRate int
	Speed        int
	Size         int
	From         string
	Status       string

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
		return fmt.Sprintf("%d%", item.ProgressRate)
	case 3:
		return fmt.Sprintf("%s/s", ByteViewLite(int64(item.Speed)))
	case 4:
		return ByteView(int64(item.Size))
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

var jobTable *JobModel

func JobTalbeUpdate()  {
	item := make([]*JobItem, 0)
	for i:= 0 ; i < 10; i++ {
		item = append(item, &JobItem{
			Index: i,
			Title: GetTimeStamp(),
			ProgressRate: rand.Int() % 100,
			Speed: rand.Int() % 1048576 ,
			Size: rand.Int() ,
			From: "youtube.com",
			Status: "ready",
		})
	}

	jobTable = new(JobModel)
	jobTable.items = item
	jobTable.PublishRowsReset()
	jobTable.Sort(jobTable.sortColumn, jobTable.sortOrder)
}

func TableWight() Widget {
	JobTalbeUpdate()
	return TableView{
		AlternatingRowBG: true,
		ColumnsOrderable: true,
		CheckBoxes: true,
		Columns: []TableViewColumn{
			{Title: "#", Width: 30},
			{Title: LangValue("title"), Width: 160},
			{Title: LangValue("progressrate")},
			{Title: LangValue("speed"), Width: 80},
			{Title: LangValue("size"), Width: 80},
			{Title: LangValue("from"), Width: 100},
			{Title: LangValue("status"), Width: 80},
		},
		StyleCell: func(style *walk.CellStyle) {
			if style.Row()%2 == 0 {
				style.BackgroundColor = walk.RGB(248, 248, 255)
			} else {
				style.BackgroundColor = walk.RGB(220, 220, 220)
			}
		},
		Model:jobTable,
	}
}

