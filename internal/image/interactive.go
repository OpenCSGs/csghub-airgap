package image

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// runtimeItem is a list item representing a selectable runtime image.
type runtimeItem struct {
	ImageInfo RuntimeImageInfo
	Selected  bool
}

func (r runtimeItem) FilterValue() string {
	return fmt.Sprintf("%s %s %s %s %s",
		r.ImageInfo.Category, r.ImageInfo.EngineName,
		r.ImageInfo.Image, r.ImageInfo.ComputeType, r.ImageInfo.EngineVer)
}

// categoryHeader is a list item that displays a category header (non-selectable).
type categoryHeader string

func (c categoryHeader) FilterValue() string { return "" }

type itemDelegate struct{}

func (d itemDelegate) Height() int  { return 1 }
func (d itemDelegate) Spacing() int { return 0 }

func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}

func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	if ch, ok := listItem.(categoryHeader); ok {
		style := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).PaddingLeft(2)
		fmt.Fprint(w, style.Render("── "+string(ch)+" ──"))
		return
	}

	ri, ok := listItem.(runtimeItem)
	if !ok {
		return
	}

	ch := " "
	if ri.Selected {
		ch = "✓"
	}

	label := fmt.Sprintf("[%s] %s  %s  (%s)  [%s]",
		ch, ri.ImageInfo.EngineName, ri.ImageInfo.Image,
		ri.ImageInfo.ComputeType, ri.ImageInfo.EngineVer)

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

type imageSelectorModel struct {
	list     list.Model
	chosen   bool
	quitting bool
}

func newImageSelectorModel(images []RuntimeImageInfo, width, height int) imageSelectorModel {
	items := make([]list.Item, 0, len(images))
	for _, img := range images {
		items = append(items, runtimeItem{ImageInfo: img, Selected: false})
	}

	d := itemDelegate{}
	l := list.New(items, d, width, height)
	l.Title = "Select runtime images (space to toggle, enter to confirm)"
	l.SetShowStatusBar(false)
	l.SetShowHelp(true)
	l.SetFilteringEnabled(false) // disable filtering so space reaches delegate
	l.KeyMap.Quit.SetEnabled(false)
	l.KeyMap.ForceQuit.SetEnabled(false)

	return imageSelectorModel{list: l}
}

func (m imageSelectorModel) Init() tea.Cmd { return nil }

func (m imageSelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h := msg.Height - 4
		m.list.SetSize(msg.Width, h)

	case tea.KeyPressMsg:
		switch msg.String() {
		case "enter":
			m.chosen = true
			return m, tea.Quit
		case "q", "esc":
			m.quitting = true
			return m, tea.Quit
		case "space":
			// Toggle selection for current item (skip category headers)
			items := m.list.Items()
			if idx := m.list.Index(); idx >= 0 && idx < len(items) {
				if ri, ok := items[idx].(runtimeItem); ok {
					ri.Selected = !ri.Selected
					items[idx] = ri
					m.list.SetItems(items)
				}
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m imageSelectorModel) View() tea.View {
	if m.quitting {
		v := tea.NewView("Cancelled.\n")
		return v
	}
	v := tea.NewView(m.list.View())
	v.AltScreen = true
	return v
}

// SelectRuntimeImages shows an interactive TUI using bubbletea for the user to pick runtime images.
// Returns the list of selected image names. Returns nil if the user cancels.
func SelectRuntimeImages(ctx context.Context, source Source, version string) ([]string, error) {
	images, err := ListRuntimeImages(ctx, source, version, nil, nil)
	if err != nil {
		return nil, err
	}

	if len(images) == 0 {
		fmt.Fprintln(os.Stderr, "No runtime images found")
		return nil, nil
	}

	// Group by category
	categoryOrder := []string{"evaluation", "finetune", "inference", "job", "notebook", "space"}
	grouped := make(map[string][]runtimeItem)
	for _, img := range images {
		grouped[img.Category] = append(grouped[img.Category], runtimeItem{ImageInfo: img, Selected: false})
	}

	// Flatten with category separators
	var allItems []list.Item
	for _, cat := range categoryOrder {
		items, ok := grouped[cat]
		if !ok {
			continue
		}
		allItems = append(allItems, categoryHeader(strings.ToUpper(cat)))
		for _, item := range items {
			allItems = append(allItems, item)
		}
	}
	// Unknown categories
	for cat, items := range grouped {
		found := false
		for _, c := range categoryOrder {
			if c == cat {
				found = true
				break
			}
		}
		if !found {
			allItems = append(allItems, categoryHeader(strings.ToUpper(cat)))
			for _, item := range items {
				allItems = append(allItems, item)
			}
		}
	}

	m := newImageSelectorModel(images, 80, 24)
	m.list.SetItems(allItems)

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
		if ri, ok := it.(runtimeItem); ok && ri.Selected {
			selected = append(selected, ri.ImageInfo.Image)
		}
	}
	return selected, nil
}
