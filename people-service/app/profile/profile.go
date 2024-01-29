package profile

import (
	"errors"
	"people-service/database"
)

func ValidateProfileByAccountID(db *database.Database, profileID int, accountID int) error {
	stmt := "SELECT id FROM `sidekiq-dev`.AccountProfile WHERE id = ? AND (accountID = ? OR managedByID = ?);"
	var profiles []interface{}
	err := db.Conn.Select(&profiles, stmt, profileID, accountID, accountID)
	if err != nil {
		return err
	}
	if len(profiles) == 0 {
		return errors.New("profile not authorized")
	}
	return nil
}
