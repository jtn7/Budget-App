package main

import (
	"database/sql"
	"fmt"
	"strings"
)

type database struct {
	*sql.DB
}

// GetUpResponses retrieves the "up" responses for a given message ID.
//
// Returns a slice of user names
func (d *database) GetUpResponses(messageID string) (users []string, err error) {
	const query = "SELECT UpResponses FROM Quizes WHERE MessageID = ?"
	var usersString string
	err = d.QueryRow(query, messageID).Scan(&usersString)
	if err != nil {
		err = fmt.Errorf("ERROR: could not get \"down\" responses from \"Quizes\" table: %w", err)
		return
	}

	users = strings.Split(usersString, ",")
	return
}

// GetDownResponses retrieves the "down" responses for a given message ID.
//
// Returns a slice of user names
func (d *database) GetDownResponses(messageID string) (users []string, err error) {
	const query = "SELECT DownResponses FROM Quizes WHERE MessageID = ?"
	var usersString string
	err = d.QueryRow(query, messageID).Scan(&usersString)
	if err != nil {
		err = fmt.Errorf("ERROR: could not get \"down\" responses from \"Quizes\" table: %w", err)
		return
	}

	users = strings.Split(usersString, ",")
	return
}

// UpdateUpResponses updates the "up" responses for a given message ID.
func (d *database) UpdateUpResponses(messageID string, users []string) (err error) {
	const query = "UPDATE Quizes (MessageID, UpResponses) VALUES (?, ?)"
	_, err = d.Exec(query, messageID)
	if err != nil {
		err = fmt.Errorf("ERROR: could not update \"up\" responses: %w", err)
		return
	}
	return
}

// UpdateDownResponses updates the "down" responses for a given message ID.
func (d *database) UpdateDownResponses(messageID string, users []string) (err error) {
	const query = "UPDATE Quizes (MessageID, DownResponses) VALUES (?, ?)"
	_, err = d.Exec(query, messageID)
	if err != nil {
		err = fmt.Errorf("ERROR: could not update \"up\" responses: %w", err)
		return
	}
	return
}
