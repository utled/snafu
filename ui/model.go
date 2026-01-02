package ui

import (
	"database/sql"
	"fmt"
	"log"
	"os/exec"
	"snafu/data"
	"snafu/search"
	"strconv"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

type Model struct {
	width        int
	height       int
	inputField   textinput.Model
	resultsTable table.Model
	previewBox   viewport.Model
	style        styles
	dbConnection *sql.DB
	err          error
}

func defaultStyles() styles {
	style := new(styles)
	style.BorderColor = lipgloss.Color("36")
	style.InputField = lipgloss.NewStyle().
		BorderForeground(style.BorderColor).
		BorderStyle(lipgloss.NormalBorder()).
		Padding(0)
	return *style
}

func NewModel(dbConn *sql.DB) Model {
	inputStyles := defaultStyles()
	textInput := textinput.New()
	textInput.Placeholder = "Enter search term"
	textInput.Width = 30
	textInput.Focus()

	columns := []table.Column{
		{Title: "Path", Width: 20},
		{Title: "Name", Width: 20},
		{Title: "Size", Width: 10},
		{Title: "Modified", Width: 20},
		{Title: "Last Used", Width: 20},
		{Title: "Metadata Changed", Width: 20},
	}

	resultTable := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(20),
	)

	tableStyle := table.DefaultStyles()
	tableStyle.Selected = tableStyle.Header.
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(false).
		Bold(true)

	tableStyle.Selected = tableStyle.Selected.
		Foreground(lipgloss.Color("36")).
		Bold(true)

	resultTable.SetStyles(tableStyle)

	return Model{
		dbConnection: dbConn,
		inputField:   textInput,
		resultsTable: resultTable,
		style:        inputStyles,
	}
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

func openFile(path string) tea.Cmd {
	return func() tea.Msg {
		var cmd *exec.Cmd
		cmd = exec.Command("xdg-open", path)
		_ = cmd.Run()
		return nil
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

func (model Model) Init() tea.Cmd {
	return textinput.Blink
}

func (model Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	oldInput := model.inputField.Value()

	var inputCmd tea.Cmd
	model.inputField, inputCmd = model.inputField.Update(msg)
	cmds = append(cmds, inputCmd)

	newInput := model.inputField.Value()
	if newInput != oldInput && newInput != "" {
		cmds = append(cmds, model.searchDB(newInput))
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		remainingHeight := msg.Height - 12
		model.width = msg.Width
		model.height = msg.Height
		model.resultsTable.SetHeight(remainingHeight / 4 * 3)
		model.previewBox.Width = msg.Width - 4
		model.previewBox.Height = remainingHeight / 4
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return model, tea.Quit
		}
		switch msg.Type {
		case tea.KeyEnter:
			selectedRow := model.resultsTable.SelectedRow()
			if selectedRow != nil {
				filePath := selectedRow[0]
				return model, openFile(filePath)
			}
		case tea.KeyUp, tea.KeyDown:
			var cmd tea.Cmd
			model.resultsTable, cmd = model.resultsTable.Update(msg)
			cmds = append(cmds, cmd)

			if len(model.resultsTable.Rows()) > 0 {
				path := model.resultsTable.SelectedRow()[0]
				cmds = append(cmds, model.fetchPreview(path))
			}
		}
	case searchResult:
		var rows []table.Row
		for _, entry := range msg.Rows {
			rows = append(rows, table.Row{
				entry.Path,
				entry.Name,
				strconv.FormatInt(entry.Size, 10),
				entry.ModificationTime.Format("2006-01-02 15:04:05"),
				entry.AccessTime.Format("2006-01-02 15:04:05"),
				entry.MetaDataChangeTime.Format("2006-01-02 15:04:05"),
			})
		}
		model.resultsTable.SetRows(rows)
		model.resultsTable.GotoTop()

		if len(rows) > 0 {
			return model, model.fetchPreview(rows[0][0])
		}
		model.previewBox.SetContent("")
		return model, nil
	case previewResult:
		wrapWidth := model.previewBox.Width - 2
		if wrapWidth <= 0 {
			wrapWidth = 1
		}

		wrappedContent := lipgloss.NewStyle().
			Width(wrapWidth).
			Render(msg.Content)

		model.previewBox.SetContent(wrappedContent)
		model.previewBox.GotoTop()
		return model, nil
	}

	return model, tea.Batch(cmds...)
}

func (model Model) View() string {
	if model.width == 0 {
		return "loading..."
	}

	selectedPath := "None"
	if len(model.resultsTable.Rows()) > 0 {
		selectedPath = model.resultsTable.SelectedRow()[0]
	}
	selectionInfo := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Italic(true).
		PaddingBottom(1).
		Render(selectedPath)

	previewStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), true, false, false, false). // Top border only
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 0).
		Width(model.width)

	return lipgloss.Place(
		model.width,
		model.height,
		lipgloss.Center,
		lipgloss.Top,
		lipgloss.JoinVertical(
			lipgloss.Left,
			model.style.InputField.Render(model.inputField.View()),
			selectionInfo,
			model.resultsTable.View(),
			previewStyle.Render(model.previewBox.View()),
			"\n "+lipgloss.NewStyle().
				Foreground(lipgloss.Color("240")).
				Render("[Arrows] Move • [Enter] Open • [Q] Quit"),
		),
	)
}
