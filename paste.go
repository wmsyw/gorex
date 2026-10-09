package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"
)

// pasteShortcut is the key that pastes into a pane: Command-V on macOS,
// Control-Shift-V elsewhere, where Control and a letter are the terminal's.
// GoRex takes it itself, so that an image on the clipboard pastes as the
// path of the file it is written to, which the programs that read images
// take, rather than as nothing at all.
func pasteShortcut() (ui.Modifiers, ui.Key) {
	if runtime.GOOS == "darwin" {
		return ui.Cmd, ui.KeyV
	}
	return ui.Ctrl | ui.Shift, ui.KeyV
}

// pasteText is what a paste sends to a program: the clipboard's text, the
// paths of the files it holds, or the path of the file its image is
// written to, in that order, and empty when it holds none of them.
func pasteText() string {
	// A view without an application, as the tests build, has no clipboard
	// to read, and MyGo's waits for a loop that is not there.
	if !mygo.App.IsReady() {
		return ""
	}
	if text, err := mygo.Clipboard.ReadFormat(transfer.Text); err == nil && len(text) > 0 {
		return string(text)
	}
	if files, err := mygo.Clipboard.ReadFiles(); err == nil && len(files) > 0 {
		for i, f := range files {
			if strings.ContainsAny(f, " \t\"'\\$&()[]{}*?!<>|;#") {
				files[i] = "'" + strings.ReplaceAll(f, "'", `'\''`) + "'"
			}
		}
		return strings.Join(files, " ")
	}
	png, err := mygo.Clipboard.ReadFormat(transfer.PNG)
	if err != nil || len(png) == 0 {
		return ""
	}
	path, err := writePastedImage(png)
	if err != nil {
		return ""
	}
	return path
}

// writePastedImage writes an image of the clipboard under the temporary
// directory, as other terminals do, and returns the path it wrote.
func writePastedImage(png []byte) (string, error) {
	dir := filepath.Join(os.TempDir(), "gorex-paste")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	path := filepath.Join(dir, fmt.Sprintf("image-%d.png", time.Now().UnixMilli()))
	if err := os.WriteFile(path, png, 0o600); err != nil {
		return "", err
	}
	return path, nil
}

// pasteInto pastes text into a pane's program, bracketed when the program
// asks for it, as the terminal's own paste does.
func (a *App) pasteInto(p *Pane, text string) {
	if p == nil || p.term == nil || text == "" {
		return
	}
	p.term.Paste(text)
}

// pasteFocused pastes the clipboard where the focus is: into the field of
// the palette or of a rename, when one is open, and into the focused
// pane's program otherwise.
func (a *App) pasteFocused() {
	text := pasteText()
	if text == "" {
		return
	}
	switch {
	case a.paletteOpen, a.renaming != nil:
		a.do(func() {
			switch {
			case a.paletteOpen:
				a.paletteQuery += oneLine(text)
			case a.renaming != nil:
				a.renameText += oneLine(text)
			}
		})
	default:
		if t := a.tab(); t != nil {
			a.pasteInto(t.Focus, text)
		}
	}
}

// oneLine makes text that a single-line field takes, its lines joined by
// spaces.
func oneLine(text string) string {
	return strings.Join(strings.Fields(text), " ")
}
