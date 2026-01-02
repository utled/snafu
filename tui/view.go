package tui

import "github.com/charmbracelet/lipgloss"

func (model Model) renderEditView() string {
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("36")).
		Padding(0).
		Width(40)

	form := lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Render("Edit Item Tags"),
		"",
		"Tags (comma separated):",
		model.tagEditField.View(),
		"",
		lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("Enter: Save • Esc: Cancel"),
	)

	return lipgloss.Place(
		model.width,
		model.height,
		lipgloss.Center,
		lipgloss.Center,
		modalStyle.Render(form),
		// This optional argument adds a background "shroud"
		lipgloss.WithWhitespaceChars("░"),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("240")),
	)
}

func (model Model) renderSearchView() string {
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

	footerText := "[Arrows] Move • [Enter] Open • [Space] Copy Path • [Q] Quit"
	copiedLabel := ""
	if model.showCopied {
		copiedLabel = " path copied to clipboard"
	}

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
				Render(footerText+"\n\n"+copiedLabel),
		),
	)
}

func (model Model) View() string {
	searchView := model.renderSearchView()
	if model.state == stateEdit {
		editView := model.renderEditView()
		return editView
	}

	return searchView
}
