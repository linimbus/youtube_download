package main

import (
	"fmt"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"sort"
	"sync"
	"time"
)

type VideoFormat struct {
	ItagNo       int
	Quality      string // tiny/small/medium/large/hd720/hd1080/
	Format       string // video/mp4、video/webm、audio/mp4
	MimeType     string // video/mp4; codecs="avc1.42001E, mp4a.40.2"
	FPS          int
	Width        int
	Height       int
	Length       int

	checked      bool
}

type VideoModel struct {
	sync.RWMutex

	update       func(v *VideoModel)

	Timestamp    string
	Title        string
	Author       string
	Duration     time.Duration

	walk.TableModelBase
	walk.SorterBase
	sortColumn int
	sortOrder  walk.SortOrder

	items      []*VideoFormat
}

func (n *VideoModel)RowCount() int {
	return len(n.items)
}

func (n *VideoModel)Value(row, col int) interface{} {
	item := n.items[row]
	switch col {
	case 0:
		return item.ItagNo
	case 1:
		return item.Quality
	case 2:
		return item.Format
	case 3:
		return item.MimeType
	case 4:
		if item.FPS == 0 {
			return "-"
		}
		return fmt.Sprintf("%d", item.FPS)
	case 5:
		if item.Width == 0 || item.Height == 0 {
			return "-"
		}
		return fmt.Sprintf("%d*%d", item.Width, item.Height)
	case 6:
		return ByteView(int64(item.Length))
	}
	panic("unexpected col")
}

func (n *VideoModel) Checked(row int) bool {
	return n.items[row].checked
}

func (n *VideoModel) SetChecked(row int, checked bool) error {
	n.items[row].checked = checked
	return nil
}

func (m *VideoModel) Sort(col int, order walk.SortOrder) error {
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
			return c(a.ItagNo < b.ItagNo)
		case 1:
			return c(a.Quality < b.Quality)
		case 2:
			return c(a.Format < b.Format)
		case 3:
			return c(a.MimeType < b.MimeType)
		case 4:
			return c(a.FPS < b.FPS)
		case 5:
			return c(a.Width*a.Height < b.Width*b.Height)
		case 6:
			return c(a.Length < b.Length)
		}
		panic("unreachable")
	})
	return m.SorterBase.Sort(col, order)
}

func (n *VideoModel) Update(video *VideoModel)  {
	n.Title = video.Title
	n.Author = video.Author
	n.Duration = video.Duration

	n.items = video.items
	n.PublishRowsReset()
	n.Sort(n.sortColumn, n.sortOrder)

	n.update(video)
}

func (n *VideoModel)Flash()  {
	n.PublishRowsReset()
	n.Sort(n.sortColumn, n.sortOrder)
}

func NewVideoMode() *VideoModel {
	video := new(VideoModel)
	video.items = make([]*VideoFormat, 0)
	return video
}

func VideoWight(video *VideoModel) []Widget {
	var tableView *walk.TableView
	var title, author, duration *walk.Label

	video.update = func(v *VideoModel) {
		title.SetText(v.Title)
		author.SetText(v.Author)
		duration.SetText(fmt.Sprintf("%v", v.Duration))
	}

	return []Widget{
		Composite{
			Layout: Grid{Columns: 2, MarginsZero: true},
			Children: []Widget{
				Label{
					Text: "Title" + ":",
				},
				Label{
					AssignTo: &title,
					Text: video.Title,
				},
				Label{
					Text: "Author" + ":",
				},
				Label{
					AssignTo: &author,
					Text: video.Author,
				},
				Label{
					Text: "Duration" + ":",
				},
				Label{
					AssignTo: &duration,
					Text: func() string {
						if video.Duration == 0 {
							return ""
						}
						return fmt.Sprintf("%v", video.Duration)
					},
				},
			},
		},
		TableView{
			AssignTo: &tableView,
			AlternatingRowBG: true,
			ColumnsOrderable: true,
			CheckBoxes: true,
			Columns: []TableViewColumn{
				{Title: "No.", Width: 50},
				{Title: "Quality", Width: 50},
				{Title: "Format", Width: 80},
				{Title: "MimeType", Width: 250},
				{Title: "FPS", Width: 30},
				{Title: "Screen", Width: 80},
				{Title: "Length", Width: 60},
			},
			StyleCell: func(style *walk.CellStyle) {
				//item := jobTable.items[style.Row()]
				if style.Row()%2 == 0 {
					style.BackgroundColor = walk.RGB(248, 248, 255)
				} else {
					style.BackgroundColor = walk.RGB(220, 220, 220)
				}
			},
			Model:video,
		},
	}
}

