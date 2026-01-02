package tui

import (
	"database/sql"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	dbConnection *sql.DB
	width        int
	height       int
	style        styles
	state        sessionState
	selectedIdx  int
	showCopied   bool
	inputField   textinput.Model
	tagEditField textinput.Model
	tagDetails   tagDetails
	resultsTable table.Model
	previewBox   viewport.Model
	err          error
}

func NewModel(dbConn *sql.DB) Model {
	inputStyles := defaultStyles()
	textInput := textinput.New()
	textInput.Placeholder = "Enter search term"
	textInput.Width = 30
	textInput.Focus()

	tagInput := textinput.New()
	tagInput.Placeholder = "Edit tags for entry"

	columns := []table.Column{
		{Title: "Path", Width: 20},
		{Title: "Name", Width: 20},
		{Title: "Size", Width: 10},
		{Title: "Modified", Width: 20},
		{Title: "Inode", Width: 20},
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
		tagEditField: tagInput,
		resultsTable: resultTable,
		style:        inputStyles,
	}
}

func (model Model) Init() tea.Cmd {
	return textinput.Blink
}
