package profile

import (
	"database/sql"
	"fmt"
	"people-service/app/storage"
	"people-service/consts"
	"people-service/database"
	"people-service/helper"
	"people-service/model"
	"people-service/mongodatabase"
	"people-service/util"
	"strconv"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
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

func getProfilesByUserID(db *database.Database, accountID int, storageService storage.Service) (map[string]interface{}, error) {
	var response map[string]interface{}
	data := make(map[string]interface{})

	// get number of profiles allowed for that user
	getAccStmt := "SELECT s.profiles FROM `sidekiq-dev`.Services s JOIN `sidekiq-dev`.Account u ON s.id=u.accountType WHERE u.id=?"
	var numOfProfilesAllowed *int
	err := db.Conn.Get(&numOfProfilesAllowed, getAccStmt, accountID)
	if err != nil {
		response = util.SetResponse(nil, 0, "Error processing the request.")
		return response, err
	}

	// if number of profiles based on account type is determined then fetch those profiles
	stmt := "SELECT id, firstName, lastName, screenName, defaultThingsBoard FROM `sidekiq-dev`.AccountProfile WHERE accountID = ?;"
	profiles := []*model.ConciseProfile{}
	err = db.Conn.Select(&profiles, stmt, accountID)
	if err != nil {
		response = util.SetResponse(nil, 0, "Could not fetch profiles for this user.")
		return response, err
	}
	if len(profiles) == 0 {
		data["profiles"] = profiles
		data["numOfProfiles"] = len(profiles)
		data["numOfProfilesAllowed"] = *numOfProfilesAllowed
		response = util.SetResponse(data, 1, "This user has no profiles. Please create one.")
		return response, nil
	}

	for i := range profiles {
		profiles[i].Photo, err = helper.GetProfileImage(db, storageService, accountID, profiles[i].Id)
		if err != nil {
			profiles[i].Photo = ""
			fmt.Println("unable to fetch profile photo")
		}

		profiles[i].Thumbs, err = helper.GetProfileImageThumb(db, storageService, accountID, profiles[i].Id)
		if err != nil {
			profiles[i].Thumbs = model.Thumbnails{}
			fmt.Println("unable to fetch profile photo thumb")
		}
	}

	data["profiles"] = profiles
	data["numOfProfiles"] = len(profiles)
	data["numOfProfilesAllowed"] = *numOfProfilesAllowed

	response = util.SetResponse(data, 1, "Profiles fetched successfully.")
	return response, nil
}

func getProfileCountByUserID(db *database.Database, accountID int) (map[string]interface{}, error) {
	var response map[string]interface{}
	data := make(map[string]interface{})

	// get number of profiles allowed for that user
	getAccStmt := "SELECT s.profiles FROM `sidekiq-dev`.Services s JOIN `sidekiq-dev`.Account u ON s.id=u.accountType WHERE u.id=?"
	var numOfProfilesAllowed *int
	err := db.Conn.Get(&numOfProfilesAllowed, getAccStmt, accountID)
	if err != nil {
		response = util.SetResponse(nil, 0, "Error processing the request.")
		return response, err
	}

	// if number of profiles based on account type is determined then fetch those profiles
	stmt := "SELECT COUNT(*) FROM `sidekiq-dev`.AccountProfile WHERE accountID = ?;"
	var profiles int
	err = db.Conn.Get(&profiles, stmt, accountID)
	if err != nil {
		response = util.SetResponse(nil, 0, "Could not fetch profiles for this user.")
		return response, err
	}
	if profiles == 0 {
		data["numOfProfiles"] = profiles
		data["numOfProfilesAllowed"] = *numOfProfilesAllowed
		response = util.SetResponse(data, 1, "This user has no profiles. Please create one.")
		return response, nil
	}

	data["numOfProfiles"] = profiles
	data["numOfProfilesAllowed"] = *numOfProfilesAllowed

	response = util.SetResponse(data, 1, "Profile count fetched successfully")
	return response, nil
}

func addProfile(db *database.Database, profile model.Profile, accountID int) (map[string]interface{}, error) {
	stmt := "SELECT accountType FROM `sidekiq-dev`.Account WHERE id = ?;"
	user := &model.Account{}

	err := db.Conn.Get(user, stmt, accountID)
	if err != nil {
		return nil, err
	}
	var limit int64
	switch user.AccountType {
	case 1:
		limit = 1
	case 2:
		limit = 3
	case 3:
		limit = 100
	}

	var count *int64
	fetchstmt := "SELECT COUNT(*) FROM `sidekiq-dev`.AccountProfile WHERE accountID = ?"
	err = db.Conn.Get(&count, fetchstmt, accountID)
	if err != nil {
		fmt.Println("Error herer", err)
		return nil, err
	}

	fmt.Println((*count), limit)

	if (*count) > limit {
		return nil, consts.ProfileLimitError
	}

	profile.AccountID = accountID

	stmt = `INSERT INTO ` + "`sidekiq-dev`.AccountProfile" +
		` (
				id, accountID, defaultThingsBoard, screenName, 
				firstName, lastName, phone1, phone2, email1, 
				email2, birthday, address1, address2, city, 
				state, zip, country, bio, gender, notes
			) 
			VALUES
			 (
					:id, :accountID, :defaultThingsBoard, :screenName, 
					:firstName, :lastName, :phone1, :phone2, :email1, 
					:email2,:birthday, :address1, :address2, :city, 
					:state, :zip, :country, :bio, :gender, :notes
			)`

	r, err := db.Conn.NamedExec(stmt, profile)
	resData := make(map[string]interface{})
	if err != nil {
		return nil, errors.Wrap(err, "unable to insert profile")
	}
	resData["id"], err = r.LastInsertId()
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch id of inserted profile")
	}

	stmt = "INSERT INTO `sidekiq-dev`.NotificationSettings " +
		`(
			isAllNotifications, isChatMessage, isMention, isInvite, 
			isBoardJoin, isComment, isReaction, profileID) VALUES 
			(
				true, true, true, true, 
				true, true, true, :id
			);`
	_, err = db.Conn.NamedExec(stmt, resData)
	if err != nil {
		return nil, errors.Wrap(err, "unable to insert notification settings")
	}

	stmt = "INSERT INTO `sidekiq-dev`.ShareableSettings " +
		`(
			firstName, lastName, screenName, email, 
			phone, bio, address1, birthday, 
			gender, address2, profileID) VALUES 
			(
				true, true, false, false, 
				false, false, false, false, 
				false, false, :id
			);`
	_, err = db.Conn.NamedExec(stmt, resData)
	if err != nil {
		return nil, errors.Wrap(err, "unable to insert shareable settings")
	}
	return util.SetResponse(resData, 1, "Profile Inserted Successfully"), nil
}

func getProfileInfo(db *database.Database, profileID int, storageService storage.Service) (map[string]interface{}, error) {
	stmt := `SELECT
		id, accountID, firstName, lastName, screenName, gender, visibility, shareable, searchable, followMe,
		showConnections, showBoards, showThingsFollowed, approveGroupMemberships, createDate, modifiedDate, isActive, defaultThingsBoard,
		IFNULL(screenName, "") as screenName,
		IFNULL(bio, "") as bio,
		IFNULL(email1, "") as email1,
		IFNULL(email2, "") as email2,
		IFNULL(phone1, "") as phone1,
		IFNULL(phone2, "") as phone2,
		IFNULL(address1, "") as address1,
		IFNULL(address2, "") as address2,
		IFNULL(notes, "") as notes,
		IFNULL(city, "") as city,
		IFNULL(state, "") as state,
		IFNULL(zip, "") as zip,
		IFNULL(country, "") as country,
		IFNULL(timeZone, "") as timeZone,
		IFNULL(notificationsFromTime, "") as notificationsFromTime,
		IFNULL(notificationsToTime, "") as notificationsToTime,
		IFNULL(managedByID, 0) as managedByID,
		IFNULL(tags, "") as tags,
		IFNULL(sharedInfo, "") as sharedInfo,
		IFNULL(deleteDate, CURRENT_TIMESTAMP) as deleteDate,
		IFNULL(birthday, "") as birthday,
		IFNULL(connectCodeExpiration, "") as connectCodeExpiration,
		IFNULL(notes, "") as notes

		FROM` + "`sidekiq-dev`.AccountProfile WHERE id = ?"

	profile := model.Profile{}
	err := db.Conn.Get(&profile, stmt, profileID)
	if err != nil {
		return nil, err
	}
	profile.TagsArr = strings.Split(profile.Tags, ",")

	// fetch notification settings for that profile
	notification := model.NotificationSettings{}
	stmt = "SELECT isAllNotifications, isChatMessage, isMention, isInvite, isBoardJoin, isComment, isReaction FROM `sidekiq-dev`.NotificationSettings WHERE profileID = ?"
	err = db.Conn.Get(&notification, stmt, profileID)
	if err == sql.ErrNoRows {
		profile.NotificationSettings = model.NotificationSettings{}
	} else if err != nil {
		return nil, err
	}
	profile.NotificationSettings = notification

	// if co-manager exists
	if profile.ManagedByID != 0 {
		stmt = "SELECT id, firstName, lastName, IFNULL(photo, '') AS photo, email, phone FROM `sidekiq-dev`.Account WHERE id = ?"
		err := db.Conn.Get(&profile.CoManager, stmt, profile.ManagedByID)
		if err == sql.ErrNoRows {
			profile.CoManager = model.ConciseProfile{}
		} else if err != nil {
			return nil, err
		}
	}

	// fetch shareable settings of that profile
	shareableSettings := model.ShareableSettings{}
	shareableStmt := "SELECT firstName, lastName, screenName, email, phone, bio, gender, birthday, address1, address2 FROM `sidekiq-dev`.ShareableSettings WHERE profileID = ?"
	err = db.Conn.Get(&shareableSettings, shareableStmt, profileID)
	if err == sql.ErrNoRows {
		profile.ShareableSettings = model.ShareableSettings{}
	} else if err != nil {
		return nil, err
	}
	profile.ShareableSettings = shareableSettings

	profile.Photo, err = helper.GetProfileImage(db, storageService, 0, profileID)
	if err != nil {
		profile.Photo = ""
	}

	profile.Thumbs, err = helper.GetProfileImageThumb(db, storageService, 0, profileID)
	if err != nil {
		profile.Thumbs = model.Thumbnails{}
	}

	profile.Thumbs.Original = profile.Photo

	return util.SetResponse(profile, 1, "Profile information fetched successfully."), nil
}

func editProfile(db *database.Database, profile model.Profile) (map[string]interface{}, error) {
	var count *int64

	fetchstmt := "SELECT COUNT(*) AS COUNT FROM `sidekiq-dev`.AccountProfile WHERE id = ?"
	err := db.Conn.Get(&count, fetchstmt, profile.ID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch count for email")
	}
	if *count == 0 {
		return util.SetResponse(nil, 0, "Profile does not exists. Please create one."), nil
	}
	if profile.ConnectCodeExpiration == "" {
		profile.ConnectCodeExpiration = "1w"
	}
	stmt := `UPDATE` + "`sidekiq-dev`.AccountProfile" + `
			SET
				screenName = :screenName,
				firstName = :firstName,
				lastName = :lastName,
				phone1 = :phone1,
				phone2 = :phone2,
				email1 = :email1,
				email2 = :email2, 
				gender = :gender,
				address1 = :address1, 
				address2 = :address2,
				bio = :bio,
				birthday = :birthday,
				city = :city,
				state = :state,
				zip = :zip,
				country = :country,
				connectCodeExpiration = :connectCodeExpiration
			WHERE
				id = :id`
	_, err = db.Conn.NamedExec(stmt, profile)
	if err != nil {
		return nil, errors.Wrap(err, "error in updating Profiles")
	}

	return util.SetResponse(nil, 1, "Profile Updated Successfully"), nil
}

func fetchTags(db *database.Database, profileID int) (map[string]interface{}, error) {
	stmt := "SELECT tags FROM `sidekiq-dev`.AccountProfile WHERE id = ?"

	var profileTags string
	err := db.Conn.Get(&profileTags, stmt, profileID)
	if err != nil {
		return nil, err
	}

	if len(profileTags) == 0 {
		return util.SetResponse(nil, 1, "Profile has no tags"), nil
	}
	return util.SetResponse(profileTags, 1, "Tags fetched successfully."), nil
}

func updateProfileTagsNew(db *mongodatabase.DBConfig, mySql *database.Database, profileID string) error {
	profileIDInt, _ := strconv.Atoi(profileID)
	var profileTags []string

	// fetch all tags from mongo collections where owner is profileID
	// errCh := make(chan error)
	// // filter := bson.D{{Key: "owner", Value: profileID}}
	// filter := bson.M{"$and": bson.A{
	// 	bson.M{"owner": profileID},
	// 	bson.M{"state": "ACTIVE"},
	// }}
	// var mx sync.Mutex

	// // fetch all Notes tags and append to tags
	// go func(errChan chan<- error) {
	// 	defer util.RecoverGoroutinePanic(errChan)
	// 	var notes []*model.Note
	// 	noteConn, err := db.New(consts.Note)
	// 	if err != nil {
	// 		fmt.Println("unable to connect note")
	// 		errCh <- errors.Wrap(err, "unable to connect note")
	// 		return
	// 	}
	// 	noteCollection, noteClient := noteConn.Collection, noteConn.Client
	// 	defer noteClient.Disconnect(context.TODO())
	// 	curr, err := noteCollection.Find(context.TODO(), filter)
	// 	if err != nil {
	// 		fmt.Println("unable to fetch note tags")
	// 		errCh <- errors.Wrap(err, "unable to fetch note tags")
	// 		return
	// 	}
	// 	err = curr.All(context.TODO(), &notes)
	// 	if err != nil {
	// 		fmt.Println("error while note")
	// 		errCh <- errors.Wrap(err, "failure in variable mapping")
	// 		return
	// 	}
	// 	for i := range notes {
	// 		mx.Lock()
	// 		profileTags = append(profileTags, notes[i].Tags...)
	// 		mx.Unlock()
	// 	}
	// 	errCh <- nil
	// }(errCh)

	// // fetch all Tasks tags and append to tags
	// go func(errChan chan<- error) {
	// 	defer util.RecoverGoroutinePanic(errChan)
	// 	var tasks []*model.Task
	// 	taskConn, err := db.New(consts.Task)
	// 	if err != nil {
	// 		fmt.Println("unable to connect task")
	// 		errCh <- errors.Wrap(err, "unable to connect task")
	// 		return
	// 	}
	// 	taskCollection, taskClient := taskConn.Collection, taskConn.Client
	// 	defer taskClient.Disconnect(context.TODO())
	// 	curr, err := taskCollection.Find(context.TODO(), filter)
	// 	if err != nil {
	// 		fmt.Println("unable to fetch task tags")
	// 		errCh <- errors.Wrap(err, "unable to fetch task tags")
	// 		return
	// 	}
	// 	err = curr.All(context.TODO(), &tasks)
	// 	if err != nil {
	// 		fmt.Println("error while task")
	// 		errCh <- errors.Wrap(err, "failure in variable mapping")
	// 		return
	// 	}
	// 	// for i := range tasks {
	// 	// 	mx.Lock()
	// 	// 	profileTags = append(profileTags, tasks[i].Tags...)
	// 	// 	mx.Unlock()
	// 	// }
	// 	errCh <- nil
	// }(errCh)

	// // fetch all Files tags and append to tags
	// go func(errChan chan<- error) {
	// 	defer util.RecoverGoroutinePanic(errChan)
	// 	var files []*model.UploadedFile
	// 	fileConn, err := db.New(consts.File)
	// 	if err != nil {
	// 		fmt.Println("unable to connect file")
	// 		errCh <- errors.Wrap(err, "unable to connect file")
	// 		return
	// 	}
	// 	fileCollection, fileClient := fileConn.Collection, fileConn.Client
	// 	defer fileClient.Disconnect(context.TODO())
	// 	curr, err := fileCollection.Find(context.TODO(), filter)
	// 	if err != nil {
	// 		fmt.Println("error while file")
	// 		fmt.Println("unable to fetch files tags")
	// 		errCh <- errors.Wrap(err, "unable to fetch files tags")
	// 		return
	// 	}
	// 	err = curr.All(context.TODO(), &files)
	// 	if err != nil {
	// 		errCh <- errors.Wrap(err, "failure in variable mapping")
	// 		return
	// 	}
	// 	for i := range files {
	// 		mx.Lock()
	// 		profileTags = append(profileTags, files[i].Tags...)
	// 		mx.Unlock()
	// 	}
	// 	errCh <- nil
	// }(errCh)

	// // fetch all Collection tags and append to tags
	// go func(errChan chan<- error) {
	// 	defer util.RecoverGoroutinePanic(errChan)
	// 	var col []*model.Collection
	// 	colConn, err := db.New(consts.Collection)
	// 	if err != nil {
	// 		fmt.Println("unable to connect collection")
	// 		errCh <- errors.Wrap(err, "unable to connect collection")
	// 		return
	// 	}
	// 	colCollection, colClient := colConn.Collection, colConn.Client
	// 	defer colClient.Disconnect(context.TODO())
	// 	curr, err := colCollection.Find(context.TODO(), filter)
	// 	if err != nil {
	// 		fmt.Println("unable to fetch collection tags")
	// 		errCh <- errors.Wrap(err, "unable to fetch collection tags")
	// 		return
	// 	}
	// 	err = curr.All(context.TODO(), &col)
	// 	if err != nil {
	// 		fmt.Println("error while Collection")
	// 		errCh <- errors.Wrap(err, "failure in variable mapping")
	// 		return
	// 	}
	// 	for i := range col {
	// 		mx.Lock()
	// 		profileTags = append(profileTags, col[i].Tags...)
	// 		mx.Unlock()
	// 	}
	// 	errCh <- nil
	// }(errCh)

	// // fetch all Board tags and append to tags
	// go func(errChan chan<- error) {
	// 	defer util.RecoverGoroutinePanic(errChan)
	// 	var boards []*model.Board
	// 	boardConn, err := db.New(consts.Board)
	// 	if err != nil {
	// 		fmt.Println("unable to connect board")
	// 		errCh <- errors.Wrap(err, "unable to connect board")
	// 		return
	// 	}
	// 	boardCollection, boardClient := boardConn.Collection, boardConn.Client
	// 	defer boardClient.Disconnect(context.TODO())
	// 	curr, err := boardCollection.Find(context.TODO(), filter)
	// 	if err != nil {
	// 		fmt.Println("unable to fetch note")
	// 		errCh <- errors.Wrap(err, "unable to fetch board tags")
	// 		return
	// 	}
	// 	err = curr.All(context.TODO(), &boards)
	// 	if err != nil {
	// 		fmt.Println("error while Board")
	// 		errCh <- errors.Wrap(err, "failure in variable mapping")
	// 		return
	// 	}
	// 	for i := range boards {
	// 		mx.Lock()
	// 		profileTags = append(profileTags, boards[i].Tags...)
	// 		mx.Unlock()
	// 	}
	// 	errCh <- nil
	// }(errCh)

	// for i := 0; i < 5; i++ {
	// 	if err := <-errCh; err != nil {
	// 		fmt.Printf("error occurred from go routine%v", err)
	// 		return err
	// 	}
	// }
	// update in mysql
	profileTags = util.RemoveArrayDuplicate(profileTags)
	profileTagsStr := strings.Join(profileTags, ",")
	p := model.Profile{ID: profileIDInt, Tags: profileTagsStr}
	updateStmt := "UPDATE `sidekiq-dev`.AccountProfile SET tags = :tags WHERE id = :id"
	_, err := mySql.Conn.NamedExec(updateStmt, p)
	if err != nil {
		return errors.Wrap(err, "unable to perform update query in MySQL")
	}
	return nil
}

func deleteProfile(db *database.Database, accountID string, profileIDs []string) (map[string]interface{}, error) {
	deletedAT := time.Now()
	stmt := "UPDATE `sidekiq-dev`.AccountProfile" + `
	SET
	deleteDate = ?
	WHERE id IN (?) and accountID=?
	`
	query, args, err := sqlx.In(stmt, deletedAT, profileIDs, accountID)
	if err != nil {
		return nil, err
	}
	query = db.Conn.Rebind(query)
	result := db.Conn.MustExec(query, args...)
	in, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	fmt.Println("Rows affected", in)
	if in == 0 {
		return util.SetResponse(nil, 1, "profile does not belongs to you"), nil
	}
	return util.SetResponse(nil, 1, "Profiles deleted successfully"), nil
}
