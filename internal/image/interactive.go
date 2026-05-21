package image

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"golang.org/x/term"
)

// SelectableImage wraps RuntimeImageInfo with selection state.
type SelectableImage struct {
	RuntimeImageInfo
	Selected bool
}

// SelectRuntimeImages shows an interactive TUI for the user to pick runtime images.
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

	// Group by category, preserve order
	categoryOrder := []string{"evaluation", "finetune", "inference", "job", "notebook", "space"}
	grouped := make(map[string][]*SelectableImage)
	for i := range images {
		img := &SelectableImage{RuntimeImageInfo: images[i], Selected: false}
		grouped[img.Category] = append(grouped[img.Category], img)
	}

	// Build flat display list
	var categories []string
	for _, c := range categoryOrder {
		if _, ok := grouped[c]; ok {
			categories = append(categories, c)
		}
	}
	// Add any unknown categories
	for c := range grouped {
		found := false
		for _, known := range categories {
			if known == c {
				found = true
				break
			}
		}
		if !found {
			categories = append(categories, c)
		}
	}

	// Flatten into a single list with category markers
	type entry struct {
		isHeader bool
		category string
		img      *SelectableImage
	}

	var entries []entry
	for _, cat := range categories {
		entries = append(entries, entry{isHeader: true, category: cat})
		for _, img := range grouped[cat] {
			entries = append(entries, entry{img: img})
		}
	}

	// Enter raw terminal mode
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return nil, fmt.Errorf("cannot set raw terminal: %w", err)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	// Hide cursor
	fmt.Print("\033[?25l")
	defer fmt.Print("\033[?25h")

	scroll := 0
	cursor := 0

	render := func() {
		height := 20 // reasonable terminal lines
		if h, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && h > 5 {
			height = h - 3
		}

		// Clear screen, cursor home
		fmt.Print("\033[H\033[2J")

		fmt.Println("\033[1mSelect runtime images (Space: toggle, Enter: confirm, q/Esc: cancel)\033[0m")
		fmt.Println(strings.Repeat("─", 80))

		startIdx := 0
		// Find start index for scroll
		idx := 0
		for i, e := range entries {
			if i < cursor {
				if !e.isHeader {
					idx++
				}
			}
			if i == cursor {
				startIdx = idx
				break
			}
			if !e.isHeader {
				idx++
			}
		}

		// Adjust scroll so cursor is visible
		if startIdx < scroll {
			scroll = startIdx
		}
		if startIdx >= scroll+height {
			scroll = startIdx - height + 1
		}

		lineCount := 0
		for i, e := range entries {
			if lineCount >= height {
				break
			}
			if !e.isHeader {
				if idx < scroll {
					idx++
					continue
				}
			}

			if e.isHeader {
				catTitle := strings.ToUpper(e.category)
				if cursor == i {
					fmt.Printf("\033[1;36m▶ %s\033[0m\n", catTitle)
				} else {
					fmt.Printf("  \033[1;36m%s\033[0m\n", catTitle)
				}
				lineCount++
			} else {
				ch := " "
				if e.img.Selected {
					ch = "✗"
				}
				label := fmt.Sprintf("[%s] %s  %s  (%s)  [%s]",
					ch, e.img.EngineName, e.img.Image, e.img.ComputeType, e.img.EngineVer)
				if cursor == i {
					fmt.Printf("\033[1;33m▶ %s\033[0m\n", label)
				} else {
					fmt.Printf("  %s\n", label)
				}
				idx++
				lineCount++
			}
		}

		// Count selected
		selected := 0
		for _, e := range entries {
			if e.img != nil && e.img.Selected {
				selected++
			}
		}
		fmt.Printf("\n\033[1mSelected: %d images\033[0m (Enter to confirm, q/Esc to cancel)\n", selected)
	}

	render()

	buf := make([]byte, 3)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			return nil, err
		}

		// Handle escape sequence
		if n == 3 && buf[0] == 27 && buf[1] == 91 {
			switch buf[2] {
			case 65: // up
				for cursor > 0 {
					cursor--
					if !entries[cursor].isHeader {
						break
					}
				}
			case 66: // down
				for cursor < len(entries)-1 {
					cursor++
					if !entries[cursor].isHeader {
						break
					}
				}
			}
			render()
			continue
		}

		switch buf[0] {
		case 13, 10: // enter
			var result []string
			for _, e := range entries {
				if e.img != nil && e.img.Selected {
					result = append(result, e.img.Image)
				}
			}
			// Clear screen, show cursor
			fmt.Print("\033[H\033[2J\033[?25h")
			return result, nil

		case ' ':
			if entries[cursor].img != nil {
				entries[cursor].img.Selected = !entries[cursor].img.Selected
			}
			render()

		case 'q':
			fmt.Print("\033[H\033[2J\033[?25h")
			return nil, nil

		case 27: // escape
			fmt.Print("\033[H\033[2J\033[?25h")
			return nil, nil
		}
	}
}

// PrintRuntimeImagesTable prints all runtime images grouped by category (non-interactive).
func PrintRuntimeImagesTable(ctx context.Context, source Source, version string) error {
	images, err := ListRuntimeImages(ctx, source, version, nil, nil)
	if err != nil {
		return err
	}

	// Group by category
	categoryOrder := []string{"evaluation", "finetune", "inference", "job", "notebook", "space"}
	grouped := make(map[string][]RuntimeImageInfo)
	for _, img := range images {
		grouped[img.Category] = append(grouped[img.Category], img)
	}

	for _, cat := range categoryOrder {
		imgs, ok := grouped[cat]
		if !ok {
			continue
		}
		fmt.Printf("\n\033[1m[%s]\033[0m\n", strings.ToUpper(cat))
		for _, img := range imgs {
			fmt.Printf("  %-25s %s  (%s)  [%s]\n", img.EngineName, img.Image, img.ComputeType, img.EngineVer)
		}
	}

	// Unknown categories
	for cat, imgs := range grouped {
		found := false
		for _, c := range categoryOrder {
			if c == cat {
				found = true
				break
			}
		}
		if !found {
			sort.Slice(imgs, func(i, j int) bool {
				return imgs[i].Image < imgs[j].Image
			})
			fmt.Printf("\n\033[1m[%s]\033[0m\n", strings.ToUpper(cat))
			for _, img := range imgs {
				fmt.Printf("  %-25s %s  (%s)  [%s]\n", img.EngineName, img.Image, img.ComputeType, img.EngineVer)
			}
		}
	}

	fmt.Printf("\nTotal: %d images\n", len(images))
	return nil
}
