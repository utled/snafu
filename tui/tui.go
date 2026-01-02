package tui

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"
	"snafu/db"

	tea "github.com/charmbracelet/bubbletea"
)

func UI() {
	homePath, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	dbPath := filepath.Join(homePath, ".snafu", "snafu.db")

	con, err := db.CreateConnection(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer func(con *sql.DB) {
		err = db.CloseConnection(con)
		if err != nil {
			log.Fatal(err)
		}
	}(con)

	model := NewModel(con)
	program := tea.NewProgram(model, tea.WithAltScreen())
	_, err = program.Run()
	if err != nil {
		log.Fatal(err)
	}
}
