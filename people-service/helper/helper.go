package helper

import (
	"fmt"
	"people-service/app/storage"
	"people-service/database"
	"people-service/model"
	"people-service/util"
)

func GetAccountImage(mysql *database.Database, storageService storage.Service, accountID, profileID int) (string, error) {
	var err error
	if accountID == 0 {
		stmt := `SELECT accountID FROM` + "`sidekiq-dev`.AccountProfile WHERE id = ?"
		err = mysql.Conn.Get(&accountID, stmt, profileID)
		if err != nil {
			return "", err
		}
	}
	key := util.GetKeyForUserImage(accountID, "")
	fileName := fmt.Sprintf("%d.png", accountID)
	fileData, err := storageService.GetUserFile(key, fileName)
	if err != nil {
		return "", err
	}
	return fileData.Filename, nil
}

func GetAccountImageThumb(mysql *database.Database, storageService storage.Service, accountID int) (model.Thumbnails, error) {
	thumbTypes := []string{"sm", "ic"}
	thumbKey := util.GetKeyForUserImage(accountID, "thumbs")
	thumbfileName := fmt.Sprintf("%d.png", accountID)
	thumbs, err := GetThumbnails(storageService, thumbKey, thumbfileName, thumbTypes)
	if err != nil {
		thumbs = model.Thumbnails{}
	}

	return thumbs, nil
}

func GetProfileImage(mysql *database.Database, storageService storage.Service, accountID, profileID int) (string, error) {
	var err error
	if accountID == 0 {
		stmt := `SELECT accountID FROM` + "`sidekiq-dev`.AccountProfile WHERE id = ?"
		err = mysql.Conn.Get(&accountID, stmt, profileID)
		if err != nil {
			return "", err
		}
	}

	key := util.GetKeyForProfileImage(accountID, profileID, "")
	fileName := fmt.Sprintf("%d.png", profileID)
	fileData, err := storageService.GetUserFile(key, fileName)
	if err != nil {
		fmt.Println("unable to fetch profile image", err)
		return "", nil
	}
	if fileData == nil {
		return "", nil
	}
	return fileData.Filename, nil
}

func GetProfileImageThumb(mysql *database.Database, storageService storage.Service, accountID, profileID int) (model.Thumbnails, error) {
	var err error
	if accountID == 0 {
		stmt := `SELECT accountID FROM` + "`sidekiq-dev`.AccountProfile WHERE id = ?"
		err = mysql.Conn.Get(&accountID, stmt, profileID)
		if err != nil {
			return model.Thumbnails{}, err
		}
	}

	thumbTypes := []string{"sm", "ic"}
	thumbKey := util.GetKeyForProfileImage(accountID, profileID, "thumbs")
	thumbfileName := fmt.Sprintf("%d.png", profileID)
	thumbs, err := GetThumbnails(storageService, thumbKey, thumbfileName, thumbTypes)
	if err != nil {
		thumbs = model.Thumbnails{}
	}

	return thumbs, nil
}
