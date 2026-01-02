package tui

import (
	"strconv"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (model Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch model.state {
	case stateEdit:
		var cmd tea.Cmd

		if key, ok := msg.(tea.KeyMsg); ok {
			if key.Type == tea.KeyEnter {
				model.state = stateSearch
				if model.tagEditField.Value() == "" && model.tagDetails.exists {
					return model, model.deleteTag()
				} else if !model.tagDetails.exists {
					return model, model.writeTag()
				} else {
					return model, model.updateTag()
				}
			}
			if key.Type == tea.KeyEsc {
				model.state = stateSearch // Cancel
				return model, nil
			}
		}
		model.tagEditField, cmd = model.tagEditField.Update(msg)
		return model, cmd
	case stateSearch:
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
					inode, _ := strconv.Atoi(selectedRow[4])
					return model, model.fetchTags(inode)
				}
			case tea.KeySpace:
				selectedRow := model.resultsTable.SelectedRow()
				if selectedRow != nil {
					filePath := selectedRow[0]
					return model, copyToClipboard(filePath)
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
		case copiedMsg:
			model.showCopied = true
			return model, clearCopiedAfter(time.Second * 2)
		case clearMsg:
			model.showCopied = false
			return model, nil
		case searchResult:
			var rows []table.Row
			for _, entry := range msg.Rows {
				rows = append(rows, table.Row{
					entry.Path,
					entry.Name,
					strconv.Itoa(int(entry.Size)),
					entry.ModificationTime.Format("2006-01-02 15:04:05"),
					strconv.Itoa(int(entry.Inode)),
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
		case tagFetchResult:
			model.state = stateEdit
			model.tagDetails.inode = msg.inode
			model.tagDetails.exists = msg.exists
			model.tagEditField.SetValue(msg.tags)
			return model, model.tagEditField.Focus()
		}

		return model, tea.Batch(cmds...)
	}

	return model, nil
}
