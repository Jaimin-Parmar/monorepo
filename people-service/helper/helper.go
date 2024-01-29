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
