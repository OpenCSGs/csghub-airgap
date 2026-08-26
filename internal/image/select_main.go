package image

import (
	"fmt"
	"io"
	"os"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// mainImageItem is a list item representing a selectable main chart image
// reference (e.g. opencsg-registry.cn-beijing.cr.aliyuncs.com/opencsghq/postgres:15.18).
type mainImageItem struct {
	Ref      string
	Selected bool
}

func (m mainImageItem) FilterValue() string { return m.Ref }

func (m mainImageItem) SetSelected(b bool) selectable { m.Selected = b; return m }
func (m mainImageItem) IsSelected() bool              { return m.Selected }

// mainItemDelegate renders a single main image ref with a selection marker.
type mainItemDelegate struct{}

func (d mainItemDelegate) Height() int  { return 1 }
func (d mainItemDelegate) Spacing() int { return 0 }

func (d mainItemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

func (d mainItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	mi, ok := listItem.(mainImageItem)
	if !ok {
		return
	}

	ch := " "
	if mi.Selected {
		ch = "✓"
	}

	label := fmt.Sprintf("[%s] %s", ch, mi.Ref)
	style := lipgloss.NewStyle().PaddingLeft(2)

	if index == m.Index() {
		style = style.Bold(true).Foreground(lipgloss.Color("229")).
			BorderLeft(true).BorderLeftForeground(lipgloss.Color("229"))
		label = "▸ " + label
	} else {
		style = style.Foreground(lipgloss.Color("245"))
	}

	width := m.Width()
	if width > 0 && len(label) > width-4 {
		label = label[:width-7] + "..."
	}
	fmt.Fprint(w, style.Render(label))
}

// newMainImageModel builds a list model for the given image refs.
func newMainImageModel(chartName string, refs []string, width, height int) imageSelectorModel {
	items := make([]list.Item, 0, len(refs))
	for _, ref := range refs {
		items = append(items, mainImageItem{Ref: ref, Selected: false})
	}

	d := mainItemDelegate{}
	l := list.New(items, d, width, height)
	l.Title = fmt.Sprintf("Select %s main images (space=toggle, a=all, A=none, enter=confirm, empty=skip)", chartName)
	l.SetShowStatusBar(false)
	l.SetShowHelp(true)
	l.SetFilteringEnabled(false)
	l.KeyMap.Quit.SetEnabled(false)
	l.KeyMap.ForceQuit.SetEnabled(false)
	l.KeyMap.NextPage.SetEnabled(false)
	l.KeyMap.PrevPage.SetEnabled(false)
	l.KeyMap.GoToStart.SetEnabled(false)
	l.KeyMap.GoToEnd.SetEnabled(false)

	return imageSelectorModel{list: l}
}

// SelectMainImages shows an interactive TUI for the user to pick a subset of
// the given main image refs of a specific chart. Returns the selected refs,
// or nil when the user cancels (esc/q).
func SelectMainImages(chartName string, refs []string) ([]string, error) {
	if len(refs) == 0 {
		fmt.Fprintln(os.Stderr, "No main images found")
		return nil, nil
	}

	m := newMainImageModel(chartName, refs, 100, 24)
	p := tea.NewProgram(m)
	result, err := p.Run()
	if err != nil {
		return nil, err
	}

	mod := result.(imageSelectorModel)
	if mod.quitting {
		return nil, nil
	}

	var selected []string
	for _, it := range mod.list.Items() {
		if mi, ok := it.(mainImageItem); ok && mi.Selected {
			selected = append(selected, mi.Ref)
		}
	}
	return selected, nil
}
