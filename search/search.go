package search

import (
	"database/sql"
	"fmt"
	"snafu/data"
)

func Index(con *sql.DB, searchString string) (searchResults []data.SearchResult, err error) {
	var query string
	var response *sql.Rows

	query = `select path, name, size, modification_time, inode 
				from entries 
				where name like ?;`
	response, err = con.Query(query, searchString+"%")
	for response.Next() {
		var entry data.SearchResult
		err = response.Scan(
			&entry.Path,
			&entry.Name,
			&entry.Size,
			&entry.ModificationTime,
			&entry.Inode,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize search results: %v", err)
		}
		searchResults = append(searchResults, entry)
	}
	if err = response.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate through search results: %v", err)
	}

	return searchResults, nil
}

func ContentSnippet(con *sql.DB, pathString string) (contentSnippet string, err error) {
	var result sql.NullString
	query := "SELECT content_snippet FROM entries WHERE path = ? LIMIT 1"
	err = con.QueryRow(query, pathString).Scan(&result)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("failed to scan content snippet: %v", err)
	}

	if result.Valid {
		contentSnippet = result.String
	}

	return contentSnippet, nil
}

func Tags(con *sql.DB, inode int) (exists bool, tags string, err error) {
	var result sql.NullString
	query := "SELECT tags FROM tagged_entries WHERE inode = ?"
	err = con.QueryRow(query, inode).Scan(&result)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, "", nil
		}
		return false, "", fmt.Errorf("failed to scan tagged entry: %v", err)
	}

	if result.Valid {
		tags = result.String
	}

	return true, tags, nil
}
