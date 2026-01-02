package tui

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"snafu/data"
	"snafu/search"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type sessionState int

const (
	stateSearch sessionState = iota
	stateEdit
)

type styles struct {
	BorderColor lipgloss.Color
	InputField  lipgloss.Style
}

type searchResult struct {
	Rows []data.SearchResult
	Err  error
}

type previewResult struct {
	Content string
}

type tagFetchResult struct {
	exists bool
	tags   string
	inode  int
}

type tagDetails struct {
	inode  int
	exists bool
}

type tagUpdateMsg struct {
	err error
}

type copiedMsg struct{}

type clearMsg struct{}

func defaultStyles() styles {
	style := new(styles)
	style.BorderColor = lipgloss.Color("36")
	style.InputField = lipgloss.NewStyle().
		BorderForeground(style.BorderColor).
		BorderStyle(lipgloss.NormalBorder()).
		Padding(0)
	return *style
}

func (model Model) fetchPreview(path string) tea.Cmd {
	return func() tea.Msg {
		content, err := search.ContentSnippet(model.dbConnection, path)
		if err != nil {
			log.Fatal(err)
		}
		return previewResult{Content: content}
	}
}

func (model Model) fetchTags(inode int) tea.Cmd {
	return func() tea.Msg {
		exists, tags, err := search.Tags(model.dbConnection, inode)
		if err != nil {
			return nil
		}
		return tagFetchResult{exists: exists, tags: tags, inode: inode}
	}
}

func clearCopiedAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return clearMsg{}
	})
}

func copyToClipboard(path string) tea.Cmd {
	return func() tea.Msg {
		binaries := []struct {
			name string
			args []string
		}{
			{"wl-copy", []string{}},             // Wayland
			{"xclip", []string{"-sel", "clip"}}, // X11
			{"xsel", []string{"-b", "-i"}},      // X11 alternative
		}

		for _, bin := range binaries {
			if _, err := exec.LookPath(bin.name); err == nil {
				cmd := exec.Command(bin.name, bin.args...)
				in, _ := cmd.StdinPipe()
				if err := cmd.Start(); err == nil {
					_, _ = io.WriteString(in, path)
					in.Close()
					_ = cmd.Wait()
					return copiedMsg{}
				}
			}
		}

		b64 := base64.StdEncoding.EncodeToString([]byte(path))
		fmt.Fprintf(os.Stdout, "\x1b]52;c;%s\x07", b64)

		return copiedMsg{}
	}
}

func (model Model) searchDB(searchString string) tea.Cmd {
	return func() tea.Msg {
		results, err := search.Index(model.dbConnection, searchString)
		if err != nil {
			fmt.Println(err)
		}
		return searchResult{Rows: results, Err: nil}
	}
}

func (model Model) writeTag() tea.Cmd {
	return func() tea.Msg {
		statement := "insert into tagged_entries(inode, tags) values(?, ?)"
		_, err := model.dbConnection.Exec(statement, model.tagDetails.inode, model.tagEditField.Value())
		model.tagDetails.exists = false
		model.tagDetails.inode = 0
		model.tagEditField.SetValue("")
		return tagUpdateMsg{err: err}
	}
}

func (model Model) updateTag() tea.Cmd {
	return func() tea.Msg {
		statement := "update tagged_entries set tags=? where inode=?"
		_, err := model.dbConnection.Exec(statement, model.tagEditField.Value(), model.tagDetails.inode)
		model.tagDetails.exists = false
		model.tagDetails.inode = 0
		model.tagEditField.SetValue("")
		return tagUpdateMsg{err: err}
	}
}

func (model Model) deleteTag() tea.Cmd {
	return func() tea.Msg {
		statement := "delete from tagged_entries where inode = ?"
		_, err := model.dbConnection.Exec(statement, model.tagDetails.inode)
		model.tagDetails.exists = false
		model.tagDetails.inode = 0
		model.tagEditField.SetValue("")
		return tagUpdateMsg{err: err}
	}
}
