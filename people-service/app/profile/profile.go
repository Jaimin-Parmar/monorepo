package profile

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"people-service/app/email"
	"people-service/app/storage"
	"people-service/consts"
	"people-service/database"
	"people-service/helper"
	"people-service/model"
	"people-service/mongodatabase"
	"people-service/util"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/pkg/errors"
	"github.com/skip2/go-qrcode"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
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

func generateCode(db *mongodatabase.DBConfig, mysql *database.Database, storageService storage.Service, accountID,
	CallerprofileID int, payload model.ConnectionRequest,
) (map[string]interface{}, error) {
	var err error
	// this temp profile ID is toggling the actual profile ID in case of
	// generating code on behalf of someone
	var tempProfileID int
	payload.ID = primitive.NewObjectID()

	if payload.ProfileID != "" {
		profileInt, err := strconv.Atoi(payload.ProfileID)
		if err != nil {
			return nil, errors.Wrap(err, "str to int conversion failed on profile id code generate")
		}
		tempProfileID = profileInt
	} else {
		payload.ProfileID = strconv.Itoa(CallerprofileID)
		tempProfileID = CallerprofileID
	}

	payload.Code = util.Get8DigitCode()
	var expTime time.Time

	var codeExpirationTime string

	stmt := "SELECT connectCodeExpiration from `sidekiq-dev`.AccountProfile where id = ?"
	err = mysql.Conn.Get(&codeExpirationTime, stmt, tempProfileID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find accountID")
	}
	payload.Duration = codeExpirationTime
	payload.CreateDate = time.Now()

	switch codeExpirationTime {
	case "1d":
		expTime = payload.CreateDate.AddDate(0, 0, 1)
	case "1h":
		expTime = payload.CreateDate.Add(time.Hour * 1)
	case "1w":
		expTime = payload.CreateDate.AddDate(0, 0, 7)
	case "1m":
		expTime = payload.CreateDate.AddDate(0, 1, 0)
	}
	payload.ExpiryDate = expTime

	cp, _ := helper.GetConciseProfile(mysql, tempProfileID, storageService)
	fileName := fmt.Sprintf("%s_%s_%s.png", cp.FirstName, cp.LastName, payload.Code)
	localQrPath := fmt.Sprintf("./%s", fileName)

	// generate QR CODE
	err = qrcode.WriteFile(payload.Code, qrcode.Medium, 256, localQrPath)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create QR code")
	}

	qrFile, err := os.Open(localQrPath)
	if err != nil {
		return nil, errors.Wrap(err, "unable to read QR code")
	}
	defer qrFile.Close()

	qrFileStat, _ := os.Stat(localQrPath)
	awsKey := util.GetKeyForProfileQR(cp.UserID, tempProfileID)
	fullPath := fmt.Sprintf("%s%s", awsKey, fileName)

	var qrFileReader io.Reader = qrFile

	f := &model.File{
		Name:   fullPath,
		Type:   "image/png",
		Size:   qrFileStat.Size(),
		ETag:   "ljksdfajklfj2l3kj4klfksjfd4llkj",
		Reader: qrFileReader,
	}

	_, err = storageService.UploadUserFile("", awsKey, fileName, f, nil, nil, nil, nil, true)
	if err != nil {
		return nil, errors.Wrap(err, "unable to upload QR")
	}

	res, err := storageService.GetUserFile(awsKey, fileName)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get presigned URL")
	}

	payload.QR = res.Filename

	err = os.Remove(localQrPath)
	if err != nil {
		return nil, errors.Wrap(err, "unable to delete qr locally.")
	}

	dbconn, err := db.New(consts.Request)
	if err != nil {
		return nil, errors.Wrap(err, "unable to establish connection with Request.")
	}
	conn, client := dbconn.Collection, dbconn.Client
	defer client.Disconnect(context.TODO())

	_, err = conn.InsertOne(context.TODO(), payload)
	if err != nil {
		return nil, errors.Wrap(err, "unable to insert connection request at mongo.")
	}

	return util.SetResponse(payload, 1, "Code generated successfully"), nil
}

func deleteCode(db *mongodatabase.DBConfig, mysql *database.Database, storageService storage.Service,
	accountID int, payload map[string]interface{}) (map[string]interface{}, error) {
	var err error

	dbconn, err := db.New(consts.Request)
	if err != nil {
		return nil, errors.Wrap(err, "unable to establish connection with Request.")
	}

	conn, client := dbconn.Collection, dbconn.Client
	defer client.Disconnect(context.TODO())

	var idsToDelete []string
	var reqs []model.ConnectionRequest

	data, ok := payload["codes"].([]interface{})
	if ok {
		for _, val := range data {
			idsToDelete = append(idsToDelete, val.(string))
		}
	} else {
		return nil, errors.Wrap(err, "invalid data mapping")
	}

	filter := bson.M{"code": bson.M{"$in": idsToDelete}}

	// find requests from mongo
	cur, err := conn.Find(context.TODO(), filter)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find all the requests")
	}
	err = cur.All(context.TODO(), &reqs)
	if err != nil {
		return nil, errors.Wrap(err, "unable to decode cursor")
	}

	// delete QR codes from wasabi
	for idx := range reqs {
		// get basic info
		senderProfileID, _ := strconv.Atoi(reqs[idx].ProfileID)
		cp, err := helper.GetConciseProfile(mysql, senderProfileID, storageService)
		if err != nil {
			return nil, errors.Wrap(err, "unable to find basic info")
		}
		fileName := fmt.Sprintf("%s_%s_%s.png", cp.FirstName, cp.LastName, reqs[idx].Code)
		key := util.GetKeyForProfileQR(accountID, senderProfileID)
		_, err = storageService.DeleteTempMedia(key, fileName)
		if err != nil {
			return nil, errors.Wrap(err, "unable to delete QR code from wasabi")
		}
	}

	// delete ids from mongo
	ret, err := conn.DeleteMany(context.TODO(), filter)
	if err != nil {
		return nil, errors.Wrap(err, "unable to delete multiple QR codes")
	}

	msg := "%s deleted successfully"
	if ret.DeletedCount == 1 {
		msg = fmt.Sprintf(msg, "Code")
	} else {
		msg = fmt.Sprintf(msg, "Codes")
	}
	return util.SetResponse(nil, 1, msg), nil
}

func updateProfileSettings(db *database.Database, update model.UpdateProfileSettings, profileID int) (map[string]interface{}, error) {
	var err error
	var stmt string
	if update.UpdateType == "Privacy" {
		update.Profile.ID = profileID

		stmt = `UPDATE ` + "`sidekiq-dev`.AccountProfile" + ` SET 
						managedByID = :managedByID, 
						visibility = :visibility, 
						followMe = :followMe, 
						shareable = :shareable, 
						showConnections = :showConnections, 
						showBoards = :showBoards, 
						notificationsFromTime = :notificationsFromTime, 
						notificationsToTime = :notificationsToTime 
					WHERE id = :id;`

		fmt.Println("update query")
		fmt.Println(stmt)
		fmt.Printf("409: %+v\n", update.Profile)
		_, err = db.Conn.NamedExec(stmt, update.Profile)
	} else if update.UpdateType == "Notification" {
		update.Profile.NotificationSettings.ProfileID = profileID

		stmt = `INSERT INTO ` + "`sidekiq-dev`.NotificationSettings" + `(
						isAllNotifications, isChatMessage, 
						isMention, isInvite, isBoardJoin, 
						isComment, isReaction, profileID
					) 
					VALUES 
						(
							:isAllNotifications, :isChatMessage, 
							:isMention, :isInvite, :isBoardJoin, 
							:isComment, :isReaction, :profileID
						) ON DUPLICATE KEY 
					UPDATE 
						isAllNotifications = :isAllNotifications, 
						isChatMessage = :isChatMessage, 
						isMention = :isMention, 
						isInvite = :isInvite, 
						isBoardJoin = :isBoardJoin, 
						isComment = :isComment, 
						isReaction = :isReaction;`

		_, err = db.Conn.NamedExec(stmt, update.Profile.NotificationSettings)
	} else {
		return util.SetResponse(nil, 0, "Settings type invalid"), nil
	}
	if err != nil {
		return nil, err
	}
	return util.SetResponse(nil, 1, fmt.Sprintf("%s settings updated successfully", update.UpdateType)), nil
}

func updateShareableSettings(db *database.Database, mongo *mongodatabase.DBConfig, profileID int,
	shareableSettings model.ShareableSettings) (map[string]interface{}, error) {
	updateStmt := "UPDATE `sidekiq-dev`.ShareableSettings SET " +
		`
		firstName =:firstName, 
		lastName =:lastName, 
		screenName =:screenName, 
		email =:email, 
		phone =:phone, 
		bio =:bio, 
		address1 =:address1, 
		address2 =:address2, 
		birthday =:birthday, 
		gender =:gender

		WHERE profileID = :profileID`

	shareableSettings.ProfileID = profileID
	fmt.Println(1792, shareableSettings)
	_, err := db.Conn.NamedExec(updateStmt, shareableSettings)
	if err != nil {
		return nil, errors.Wrap(err, "unable to update shareable-settings")
	}

	profile, err := fetchProfileInfoBasedOffShareableSettings(db, strconv.Itoa(profileID))
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch profile info based shareable off settings.")
	}

	dbconn, err := mongo.New(consts.Connection)
	if err != nil {
		return nil, errors.Wrap(err, "unable to establish connection with Connection.")
	}
	conn, client := dbconn.Collection, dbconn.Client
	defer client.Disconnect(context.TODO())

	var existingConns []model.Connection
	filter := bson.M{"connectionID": strconv.Itoa(profileID)}

	cur, err := conn.Find(context.TODO(), filter)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch existing connections.")
	}

	err = cur.All(context.TODO(), &existingConns)
	if err != nil {
		return nil, errors.Wrap(err, "unable to unpack into existing connections.")
	}

	for _, ec := range existingConns {
		if ec.ScreenName == "" && profile.ScreenName != "" {
			ec.ScreenName = profile.ScreenName
		}
		if ec.Bio == "" && profile.Bio != "" {
			ec.Bio = profile.Bio
		}
		if ec.Birthday == "" && profile.Birthday != "" {
			ec.Birthday = profile.Birthday
		}
		if ec.Address1 == "" && profile.Address1 != "" {
			ec.Address1 = profile.Address1
		}
		if ec.City == "" && profile.City != "" {
			ec.City = profile.City
		}
		if ec.Country == "" && profile.Country != "" {
			ec.Country = profile.Country
		}
		if ec.Email1 == "" && profile.Email1 != "" {
			ec.Email1 = profile.Email1
		}
		if ec.Gender == 0 && profile.Gender != 0 {
			ec.Gender = profile.Gender
		}
		if ec.Zip == "" && profile.Zip != "" {
			ec.Zip = profile.Zip
		}

		// save to mongo
		_, err = conn.UpdateOne(context.TODO(),
			bson.M{"connectionID": ec.ConnectionProfileID, "profileID": ec.ProfileID},
			bson.M{"$set": ec})
		if err != nil {
			return nil, errors.Wrap(err, "unable to update Connection record")
		}
	}

	return util.SetResponse(nil, 1, "Shareable settings updated successfully."), nil
}

func fetchProfileInfoBasedOffShareableSettings(db *database.Database, profileID string) (*model.Profile, error) {
	stmt := `
        SELECT p.firstName, p.lastName, 
            CASE 
                WHEN s.bio = true AND p.bio IS NOT NULL
                THEN p.bio 
                ELSE '' 
            END as bio,
            CASE
                WHEN s.email = true AND p.email1 IS NOT NULL 
                THEN p.email1 
                ELSE '' 
            END as email1,
            CASE
                WHEN s.screenName = true AND p.screenName IS NOT NULL 
                THEN p.screenName 
                ELSE '' 
            END as screenName,
            CASE
                WHEN s.phone = true AND p.phone1 IS NOT NULL 
                THEN p.phone1 
                ELSE '' 
            END as phone1,
            CASE
                WHEN s.gender = true AND p.gender IS NOT NULL 
                THEN p.gender 
                ELSE 0
            END as gender,
            CASE
                WHEN s.birthday = true AND p.birthday IS NOT NULL 
                THEN p.birthday 
                ELSE ''
            END as birthday,
            CASE
                WHEN s.address1 = true AND p.address1 IS NOT NULL 
                THEN p.address1 
                ELSE ''
            END as address1,
            CASE
                WHEN s.address2 = true AND p.address2 IS NOT NULL 
                THEN p.address2 
                ELSE ''
            END as address2
        FROM ` + "`sidekiq-dev`.AccountProfile as p " +
		"JOIN `sidekiq-dev`.ShareableSettings as s on p.id=s.profileID AND p.id = ?"

	var p model.Profile
	err := db.Conn.Get(&p, stmt, profileID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find shareable info")
	}

	return &p, nil
}

func sendCoManagerRequest(mysql *database.Database, db *mongodatabase.DBConfig,
	emailService email.Service, profileID int, connReq map[string]interface{},
) (map[string]interface{}, error) {
	var err error
	connReq["profileID"] = strconv.Itoa(profileID)

	dbconn, err := db.New(consts.Request)
	if err != nil {
		return nil, errors.Wrap(err, "unable to establish connection with ConnectionRequest.")
	}
	conn, client := dbconn.Collection, dbconn.Client
	defer client.Disconnect(context.TODO())

	// Get the document
	var connObj model.ConnectionRequest
	reqObjID, _ := primitive.ObjectIDFromHex(connReq["_id"].(string))
	err = conn.FindOne(context.TODO(), bson.M{"_id": reqObjID}).Decode(&connObj)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find the connection request")
	}

	// send email
	email := model.Email{
		Receiver: connReq["email1"].(string),
		Header:   "Sidekiq: Connection request",
		Subject:  "Connection request from sidekiq",
		TextBody: "Please use one of the following ways to connect",
		HtmlBody: fmt.Sprintf("<h2>%s</h2><br><img src='%s' width='200' height='200' alt='No QR' /><br>", connObj.Code, connObj.QR),
	}
	err = emailService.SendEmail(email)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to send email. Please enter a valid email")
	}

	return util.SetResponse(nil, 1, "Request sent"), nil
}

func acceptCoManagerRequest(mysql *database.Database, db *mongodatabase.DBConfig,
	storageService storage.Service, accountID, profileID int, code string,
) (map[string]interface{}, error) {
	var stmt string
	dbconn, err := db.New(consts.Request)
	if err != nil {
		return nil, err
	}

	connReqColl, connReqClient := dbconn.Collection, dbconn.Client
	defer connReqClient.Disconnect(context.TODO())

	// check if the code has expired or not
	var connReq model.ConnectionRequest
	err = connReqColl.FindOne(context.TODO(), bson.M{"code": code}).Decode(&connReq)
	if err != nil {
		return util.SetResponse(nil, 0, "The code has either expired or incorrect."), nil
	}

	if connReq.AssigneeID == "" {
		return util.SetResponse(nil, 0, "Only co-manager request code will be accepted."), nil
	}

	var senderUserID int

	assigneeProfileIDInt, err := strconv.Atoi(connReq.AssigneeID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert string to int")
	}
	senderProfileIDInt, err := strconv.Atoi(connReq.ProfileID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert string to int")
	}

	// get request sender's accountID
	stmt = "SELECT accountID FROM `sidekiq-dev`.AccountProfile WHERE id = ?"
	err = mysql.Conn.Get(&senderUserID, stmt, senderProfileIDInt)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find accountID")
	}

	// check if self connecting
	if profileID == assigneeProfileIDInt || accountID == senderUserID {
		return util.SetResponse(nil, 0, "You cannot self assign"), nil
	}

	// accountID of profileID should set in managedByID of connReq.ProfileID
	stmt = "UPDATE `sidekiq-dev`.AccountProfile SET managedByID =:managedByID where id =:id"

	profileIDInt, err := strconv.Atoi(connReq.AssigneeID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert string to int")
	}

	p := model.Profile{ID: profileIDInt, ManagedByID: accountID}
	_, err = mysql.Conn.NamedExec(stmt, p)
	if err != nil {
		return nil, errors.Wrap(err, "unable to update co-manager")
	}

	// If organization then create staff profile for co-manager
	var accountType int
	stmt = "SELECT accountType FROM `sidekiq-dev`.Account WHERE id = ?"
	err = mysql.Conn.Get(&accountType, stmt, senderUserID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find accountType")
	}
	fmt.Println("Account type", accountType)
	if accountType == 3 {
		// fetch profileID
		var profileOwnerID int
		stmt = "SELECT id FROM `sidekiq-dev`.AccountProfile WHERE managedByID = ?"
		err = mysql.Conn.Get(&profileOwnerID, stmt, accountID)
		if err != nil {
			return nil, errors.Wrap(err, "err from inserting in staff profile\nunable to find profileID")
		}
		// insert in staff profile
		parameters := map[string]interface{}{
			"id":        accountID,
			"accountID": senderUserID,
		}
		fmt.Println("These are parameters", parameters)
		stmt = "INSERT IGNORE INTO `sidekiq-dev`.OrgStaff" + `
					(accountID,managedByID,photo, firstName, lastName)
				SELECT
				:accountID,id as managedByID,photo, firstName, lastName  
				FROM` + "`sidekiq-dev`.Account" + ` WHERE id = :id;`
		_, err := mysql.Conn.NamedExec(stmt, parameters)
		if err != nil {
			return nil, errors.Wrap(err, "unable to insert in staff profile")
		}

	}

	// remove request from the collection
	_, err = connReqColl.DeleteOne(context.TODO(), bson.M{"code": code})
	if err != nil {
		return nil, errors.Wrap(err, "unable to delete the code.")
	}
	// delete QR from wasabi
	cp, _ := helper.GetConciseProfile(mysql, senderProfileIDInt, storageService)
	key := util.GetKeyForProfileQR(senderUserID, senderProfileIDInt)
	fileName := fmt.Sprintf("%s_%s_%s.png", cp.FirstName, cp.LastName, connReq.Code)
	_, err = storageService.DeleteTempMedia(key, fileName)
	if err != nil {
		return nil, errors.Wrap(err, "unable to delete QR code from wasabi")
	}
	return util.SetResponse(nil, 1, "co-manager set successfully."), nil
}

func getProfilesWithInfoByUserID(storageService storage.Service, db *database.Database, accountID int) (map[string]interface{}, error) {
	stmt := `SELECT
		id, accountID, firstName, lastName, screenName, phone1, email1, gender, visibility, shareable, searchable,
		showConnections, showBoards, showThingsFollowed, approveGroupMemberships, createDate, modifiedDate, isActive, defaultThingsBoard,
		IFNULL(photo, "") as photo,
		IFNULL(bio, "") as bio,
		IFNULL(address1, "") as address1, 
		IFNULL(address2, "") as address2,
		IFNULL(phone2, "") as phone2,
		IFNULL(email2, "") as email2,
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
		IFNULL(birthday, NOW()) as birthday,
		IFNULL(notes, "") as notes

		FROM` + "`sidekiq-dev`.AccountProfile WHERE accountID = ?"

	var profiles []model.Profile
	err := db.Conn.Select(&profiles, stmt, accountID)
	if err != nil {
		return nil, err
	}
	for i := range profiles {
		profiles[i].Photo, err = helper.GetProfileImage(db, storageService, profiles[i].AccountID, profiles[i].ID)
		if err != nil {
			profiles[i].Photo = ""
			fmt.Println("unable to fetch profile photo")
		}

		profiles[i].Thumbs, err = helper.GetProfileImageThumb(db, storageService, profiles[i].AccountID, profiles[i].ID)
		if err != nil {
			fmt.Println("unable to fetch profile thumb")
		}
	}
	return util.SetResponse(profiles, 1, "All profiles with info fetched."), nil
}

func fetchProfilesWithCoManager(storageService storage.Service, db *database.Database, mongo *mongodatabase.DBConfig, profileID int, profiles []model.Profile, search, page, limit string) (map[string]interface{}, error) {
	var err error

	dbconn, err := mongo.New(consts.Request)
	if err != nil {
		return nil, err
	}
	connReqColl, connReqClient := dbconn.Collection, dbconn.Client
	defer connReqClient.Disconnect(context.TODO())

	var filteredResults, profilesWithComanger, finalResp []model.ProfileWithCoManager

	for _, p := range profiles {
		var reqInfo model.ConnectionRequest
		pwc := model.ProfileWithCoManager{}
		pwc.ScreenName = p.ScreenName
		pwc.ID = p.ID
		pwc.Name = fmt.Sprintf("%s %s", p.FirstName, p.LastName)
		pwc.Photo = p.Photo
		pwc.Thumbs = p.Thumbs

		filter := bson.M{"profileID": strconv.Itoa(profileID), "assigneeID": strconv.Itoa(p.ID)}

		// check if code is sent or not
		count, err := connReqColl.CountDocuments(context.TODO(), filter)
		if err != nil {
			return nil, errors.Wrap(err, "unable to find count")
		}

		if count != 0 {
			err = connReqColl.FindOne(context.TODO(), filter).Decode(&reqInfo)
			if err != nil {
				return nil, err
			}
			pwc.RequestInfo = &reqInfo
		} else {
			pwc.RequestInfo = nil
		}

		if p.ManagedByID != 0 { // if managed by some user
			stmt := `SELECT IFNULL(photo, "") as comanagerPhoto, CONCAT(firstName, " ", lastName) AS comanagerName FROM ` + "`sidekiq-dev`.Account WHERE id = ?"
			err := db.Conn.Get(&pwc, stmt, p.ManagedByID)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					pwc.ManagedByID = 0
					continue
				}
				return nil, errors.Wrap(err, "unable to find co-manager")
			}
			pwc.ManagedByID = p.ManagedByID
		}
		profilesWithComanger = append(profilesWithComanger, pwc)
	}

	// search
	if search != "" {
		for _, pwc := range profilesWithComanger {
			// match search in profile name or co-manager name
			if fuzzy.Match(search, pwc.Name) || fuzzy.Match(search, pwc.CoManagerName) ||
				fuzzy.MatchFold(search, pwc.Name) || fuzzy.MatchFold(search, pwc.CoManagerName) {
				filteredResults = append(filteredResults, pwc)
			}
		}
		if len(filteredResults) == 0 {
			return util.SetPaginationResponse([]string{}, 0, 1, "No profiles with co-managers found"), nil
		}
	} else {
		filteredResults = profilesWithComanger
	}

	// pagination
	var pageNo int
	pageNo, _ = strconv.Atoi(page)
	limitInt, _ := strconv.Atoi(limit)
	var data []interface{}
	for _, d := range filteredResults {
		data = append(data, d)
	}

	subset := util.PaginateFromArray(data, pageNo, limitInt)

	for _, d := range subset {
		tmp := d.(model.ProfileWithCoManager)
		// tmp.ComanagerPhoto, err = getAccountImage(db, storageService, tmp.ManagedByID, tmp.ID)
		thumb, err := helper.GetAccountImageThumb(db, storageService, tmp.ManagedByID)
		if err != nil {
			tmp.ComanagerPhoto = ""
			fmt.Println("unable to fetch comnager profile photo")
		} else {
			tmp.ComanagerPhoto = thumb.Icon
		}

		finalResp = append(finalResp, tmp)
	}

	return util.SetPaginationResponse(finalResp, len(profilesWithComanger), 1, "Profiles with co-managers fetched successfully"), nil
}

func paginateExternalProfile(arr []model.ExternalProfile, pageNo, limit int) (ret []model.ExternalProfile) {
	var startIdx, endIdx int
	startIdx = limit * (pageNo - 1)
	endIdx = limit * pageNo

	if len(arr) == limit || len(arr) < limit {
		return arr
	}
	if endIdx < len(arr) {
		ret = arr[startIdx:endIdx]
	} else {
		ret = arr[startIdx:]
	}
	return
}

func fetchExternalProfiles(storageService storage.Service, db *database.Database, accountID int, search, page, limit string) (map[string]interface{}, error) {
	var searchFilter string
	if search != "" {
		searchFilter = ` AND
		(
			CONCAT(ap.firstName, ' ', ap.lastName) LIKE '%` + search + `%'
			OR ap.screenName LIKE '%` + search + `%'
			OR ap.firstName LIKE '%` + search + `%'
			OR ap.lastName LIKE '%` + search + `%'
		)`
	}

	managingProfiles := []model.ExternalProfile{}
	stmt := "SELECT ap.id,  ap.accountID,ap.screenName, ap.firstName, ap.lastName,ac.accountType,ac.firstName as accountFirstName,ac.lastName as accountLastName, IFNULL(ap.photo, ' ') as photo FROM `sidekiq-dev`.AccountProfile as ap INNER JOIN `sidekiq-dev`.Account as ac WHERE ap.managedByID = ? and ac.id = ap.AccountID" + searchFilter
	err := db.Conn.Select(&managingProfiles, stmt, accountID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find external profile.")
	}

	if len(managingProfiles) == 0 {
		return util.SetPaginationResponse([]string{}, 0, 1, "You are not managing any profiles."), nil
	}

	checkMap := make(map[int]string)
	for index := range managingProfiles {
		managingProfiles[index].OwnerDetails = model.OwnerDetails{}
		if managingProfiles[index].AccountType == 3 {
			name, ok := checkMap[managingProfiles[index].AccountID]
			if !ok {
				stmt = "select organizationName from `sidekiq-dev`.OrgProfile where accountID = ?"
				db.Conn.QueryRow(stmt, managingProfiles[index].AccountID).Scan(&managingProfiles[index].OwnerDetails.Name)
				checkMap[managingProfiles[index].AccountID] = managingProfiles[index].OwnerDetails.Name
				managingProfiles[index].IsOrganization = true
			} else {
				managingProfiles[index].OwnerDetails.Name = name
				managingProfiles[index].IsOrganization = true
			}
		} else {
			managingProfiles[index].IsPersonal = true
			managingProfiles[index].OwnerDetails.Name = fmt.Sprint(managingProfiles[index].AccountFirstName, " ", managingProfiles[index].AccountLastName)
		}

		managingProfiles[index].OwnerDetails.Photo, err = helper.GetProfileImage(db, storageService, managingProfiles[index].AccountID, managingProfiles[index].Id)
		if err != nil {
			managingProfiles[index].Photo = ""
			fmt.Println("unable to fetch profile photo")
		}

		managingProfiles[index].OwnerDetails.Thumbs, err = helper.GetProfileImageThumb(db, storageService, managingProfiles[index].AccountID, managingProfiles[index].Id)
		if err != nil {
			managingProfiles[index].Thumbs = model.Thumbnails{}
			fmt.Println("unable to fetch profile photo")
		}
	}

	// pagination
	var pageNo int
	pageNo, _ = strconv.Atoi(page)
	limitInt, _ := strconv.Atoi(limit)
	subset := paginateExternalProfile(managingProfiles, pageNo, limitInt)
	for i := range subset {
		if subset[i].IsPersonal {
			subset[i].Photo, err = helper.GetProfileImage(db, storageService, subset[i].AccountID, subset[i].Id)
			if err != nil {
				subset[i].Photo = ""
				fmt.Println("unable to fetch profile photo")
			}

			subset[i].Thumbs, err = helper.GetProfileImageThumb(db, storageService, subset[i].AccountID, subset[i].Id)
			if err != nil {
				subset[i].Thumbs = model.Thumbnails{}
				fmt.Println("unable to fetch profile thumb photo")
			}
		} else {
			subset[i].Photo, err = helper.GetOrgImage(db, storageService, subset[i].AccountID)
			if err != nil {
				subset[i].Photo = ""
				fmt.Println("unable to fetch org photo")
			}

			subset[i].Thumbs, err = helper.GetOrgImageThumb(db, storageService, subset[i].AccountID)
			if err != nil {
				subset[i].Photo = ""
				fmt.Println("unable to fetch org thumb photo")
			}
		}

	}
	return util.SetPaginationResponse(subset, len(managingProfiles), 1, "External profiles fetched successfully."), nil
}

func getSharableDetails(mysql *database.Database, profileID int) (*model.Profile, error) {
	var profile model.Profile

	shareableStmt := "SELECT firstName, lastName, screenName, email, phone, bio, gender, birthday, address1, address2 FROM `sidekiq-dev`.ShareableSettings WHERE profileID = ?"
	err := mysql.Conn.Get(&profile.ShareableSettings, shareableStmt, profileID)
	if err == sql.ErrNoRows {
		profile.ShareableSettings = model.ShareableSettings{}
	} else if err != nil {
		return nil, err
	}

	if profile.ShareableSettings.Bio {
		Stmt := "SELECT bio FROM `sidekiq-dev`.AccountProfile WHERE id = ?"
		err := mysql.Conn.Get(&profile, Stmt, profileID)
		if err == sql.ErrNoRows {
			profile.Bio = ""
		} else if err != nil {
			return nil, err
		}
	}

	if profile.ShareableSettings.ScreenName {
		Stmt := "SELECT screenName FROM `sidekiq-dev`.AccountProfile WHERE id = ?"
		err := mysql.Conn.Get(&profile, Stmt, profileID)
		if err == sql.ErrNoRows {
			profile.Bio = ""
		} else if err != nil {
			return nil, err
		}
	}

	return &profile, nil
}

func fetchProfileConnections(db *mongodatabase.DBConfig, mysql *database.Database, storageService storage.Service, profileID int, limit, page string, searchParameter ...string) (map[string]interface{}, error) {
	profileIDStr := strconv.Itoa(profileID)
	fmt.Println(profileIDStr)
	dbConn, err := db.New(consts.Connection)
	if err != nil {
		return nil, err
	}
	fmt.Println(profileIDStr)
	connCollection, connClient := dbConn.Collection, dbConn.Client
	defer connClient.Disconnect(context.TODO())
	pageNo, err := strconv.Atoi(page)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert string to int")
	}
	limitInt, err := strconv.Atoi(limit)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert string to int")
	}
	offset := limitInt * (pageNo - 1)

	searchPattern := fmt.Sprintf(".*%s.*", searchParameter[0])
	countPipeline := mongo.Pipeline{
		bson.D{
			{Key: "$match", Value: bson.M{
				"profileID":  profileIDStr,
				"isBlocked":  false,
				"isActive":   true,
				"isArchived": false,
			}},
		},
		bson.D{
			{Key: "$project", Value: bson.M{
				"fullName": bson.M{
					"$concat": bson.A{"$firstName", " ", "$lastName"},
				},
				"firstName":    1,
				"connectionID": 1,
				"lastName":     1,
				"screenName":   1,
				"profileID":    1,
				"relationship": 1,
				"tags":         1,
				"isBlocked":    1,
				"isActive":     1,
				"gender":       1,
			}},
		},
		bson.D{
			{Key: "$match", Value: bson.M{
				"$or": bson.A{
					bson.M{"firstName": primitive.Regex{Pattern: searchPattern, Options: "i"}},
					bson.M{"lastName": primitive.Regex{Pattern: searchPattern, Options: "i"}},
					bson.M{"fullName": primitive.Regex{Pattern: searchPattern, Options: "i"}},
					bson.M{"screenName": primitive.Regex{Pattern: searchPattern, Options: "i"}},
				},
			}},
		},
		bson.D{
			{Key: "$count", Value: "count"},
		},
	}
	pipeline := mongo.Pipeline{
		bson.D{
			{Key: "$match", Value: bson.M{
				"profileID":  profileIDStr,
				"isBlocked":  false,
				"isActive":   true,
				"isArchived": false,
			}},
		},
		bson.D{
			{Key: "$project", Value: bson.M{
				"fullName": bson.M{
					"$concat": bson.A{"$firstName", " ", "$lastName"},
				},
				"firstName":    1,
				"connectionID": 1,
				"lastName":     1,
				"screenName":   1,
				"profileID":    1,
				"relationship": 1,
				"tags":         1,
				"isBlocked":    1,
				"isActive":     1,
				"gender":       1,
			}},
		},
		bson.D{
			{Key: "$match", Value: bson.M{
				"$or": bson.A{
					bson.M{"firstName": primitive.Regex{Pattern: searchPattern, Options: "i"}},
					bson.M{"lastName": primitive.Regex{Pattern: searchPattern, Options: "i"}},
					bson.M{"fullName": primitive.Regex{Pattern: searchPattern, Options: "i"}},
					bson.M{"screenName": primitive.Regex{Pattern: searchPattern, Options: "i"}},
				},
			}},
		},
		bson.D{
			{Key: "$skip", Value: offset},
		},
		bson.D{
			{Key: "$limit", Value: limitInt},
		},
	}
	var result struct {
		Count int `bson:"count"`
	}
	cursor, err := connCollection.Aggregate(context.TODO(), countPipeline)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find count")
	}

	defer cursor.Close(context.TODO())
	if cursor.Next(context.TODO()) {
		err = cursor.Decode(&result)
		if err != nil {
			return nil, errors.Wrap(err, "unable to store in count")
		}
	}
	cursor, err = connCollection.Aggregate(context.Background(), pipeline)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find connections")
	}
	connections := make([]map[string]interface{}, 0)
	err = cursor.All(context.TODO(), &connections)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find profile's connections.")
	}

	for i := range connections {
		profileIDInt, _ := strconv.Atoi(connections[i]["connectionID"].(string))
		connections[i]["photo"], err = helper.GetProfileImage(mysql, storageService, 0, profileIDInt)
		if err != nil {
			fmt.Println("error in fetching profile photo")
		}

		connections[i]["thumbs"], err = helper.GetProfileImageThumb(mysql, storageService, 0, profileIDInt)
		if err != nil {
			fmt.Println("error in fetching profile thumb photo")
		}

		profileData, err := getSharableDetails(mysql, profileIDInt)
		if err != nil {
			fmt.Println("error in fetching profile details")
			connections[i]["screenName"] = ""
			connections[i]["bio"] = ""
		} else {
			connections[i]["screenName"] = profileData.ScreenName
			connections[i]["bio"] = profileData.Bio
		}

		connections[i]["connectionProfileID"] = connections[i]["connectionID"]
		delete(connections[i], "connectionID")
	}

	if len(connections) == 0 {
		return util.SetPaginationResponse([]int{}, int(result.Count), 1, "No active connections"), nil
	}
	return util.SetPaginationResponse(connections, int(result.Count), 1, "Connections fetched successfully"), nil
}

func fetchBoardFollowers(db *database.Database, storageService storage.Service, profileID int, limit, page string, searchParameter ...string) (map[string]interface{}, error) {
	var connections []model.FollowersInfo
	var res []model.FollowingInfo

	var err error
	stmt, boardFilter, searchFilter := "", "", ""

	// pagination calculation
	pageInt, err := strconv.Atoi(page)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert string to int")
	}
	limitInt, err := strconv.Atoi(limit)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert string to int")
	}
	offset := limitInt * (pageInt - 1)

	// adding filters if exists

	// if board ID exists
	if searchParameter[1] != "" {
		boardFilter = fmt.Sprintf("AND (boardID = '%s' OR boardTitle = '%s')", searchParameter[1], searchParameter[1])
	}

	// if any search parameter exists
	if searchParameter[0] != "" {
		searchFilter = `AND
		(
		   CONCAT(firstName, '', lastName) LIKE '%` + searchParameter[0] + `%'
		   OR screenName LIKE '%` + searchParameter[0] + `%'
		)`
	}

	// check count in db
	var count int
	stmt = `SELECT
				COUNT(DISTINCT profileID)
			FROM` +
		"`sidekiq-dev`.AccountProfile as p" + ` 
				INNER JOIN` +
		"`sidekiq-dev`.BoardsFollowed as b" + `
				ON b.profileID = p.id 
			WHERE
				p.id IN 
					(
					SELECT
						profileID 
					FROM` +
		"`sidekiq-dev`.BoardsFollowed" + `
					WHERE
						ownerID = ?  ` + boardFilter + `
					) ` + searchFilter

	err = db.Conn.Get(&count, stmt, profileID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get record's existence")
	}

	if count == 0 {
		return util.SetPaginationResponse(nil, 0, 1, "No board followers found"), nil
	}

	stmt = `SELECT
				b.boardTitle,
				b.boardID,
				p.id as connectionProfileID,
				p.firstName, 
				p.lastName,
				p.screenName 
			FROM` +
		"`sidekiq-dev`.AccountProfile as p" + ` 
				INNER JOIN` +
		"`sidekiq-dev`.BoardsFollowed as b" + `
				ON b.profileID = p.id 
			WHERE
				p.id IN 
					(
					SELECT
						profileID 
					FROM` +
		"`sidekiq-dev`.BoardsFollowed" + `
					WHERE
						ownerID = ?  ` + boardFilter + `
					) ` + searchFilter + ` LIMIT ` + limit + ` OFFSET ` + fmt.Sprintf("%v", offset)

	err = db.Conn.Select(&connections, stmt, profileID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch board followers")
	}
	for i := range connections {
		if len(res) == 0 {
			var temp model.FollowingInfo
			temp.BasicProfileInfo = connections[i].BasicProfileInfo
			temp.BoardDetails = append(temp.BoardDetails, connections[i].BoardInfo)
			res = append(res, temp)
		} else if len(res) > 0 {
			found := false
			for j := range res {
				if res[j].BasicProfileInfo.ID == connections[i].BasicProfileInfo.ID {
					res[j].BoardDetails = append(res[j].BoardDetails, connections[i].BoardInfo)
					found = true
					break
				}
			}
			if !found {
				var temp model.FollowingInfo
				temp.BasicProfileInfo = connections[i].BasicProfileInfo
				temp.BoardDetails = append(temp.BoardDetails, connections[i].BoardInfo)
				res = append(res, temp)
			}
		}
	}

	for i := range res {
		res[i].BasicProfileInfo.Photo, err = helper.GetProfileImage(db, storageService, 0, res[i].ID)
		if err != nil {
			fmt.Println("error in fetching profile photo")
		}

		res[i].BasicProfileInfo.Thumbs, err = helper.GetProfileImageThumb(db, storageService, 0, res[i].ID)
		if err != nil {
			fmt.Println("error in fetching profile thumbs photo")
		}
	}

	return util.SetPaginationResponse(res, count, 1, "Board followers fetched successfully"), nil
}

func fetchFollowingBoards(db *database.Database, storageService storage.Service, profileID int, limit, page string, searchParameter ...string) (map[string]interface{}, error) {
	var connections []model.FollowersInfo
	var res []model.FollowingInfo

	stmt, boardFilter, searchFilter := "", "", ""

	var err error

	if searchParameter[1] != "" {
		boardFilter = fmt.Sprintf("AND (b.boardID = '%s' OR b.boardTitle = '%s')", searchParameter[1], searchParameter[1])
	}

	// search filter
	if searchParameter[0] != "" {
		searchFilter = `AND
		(
		   CONCAT(firstName, '', lastName) LIKE '%` + searchParameter[0] + `%'
		   OR screenName LIKE '%` + searchParameter[0] + `%'
		)`
	}

	// pagination calculation
	pageInt, err := strconv.Atoi(page)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert string to int")
	}
	limitInt, err := strconv.Atoi(limit)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert string to int")
	}
	offset := limitInt * (pageInt - 1)

	// check count in db
	var count int
	stmt = `SELECT
				COUNT(DISTINCT  p.id)
			FROM` +
		"`sidekiq-dev`.AccountProfile as p" + `
				INNER JOIN ` +
		"`sidekiq-dev`.BoardsFollowed as b " + boardFilter + `
			WHERE
				p.id IN 
				(
				SELECT
					distinct(ownerID)
			FROM` +
		"`sidekiq-dev`.BoardsFollowed" + `
				WHERE
					profileID = ?
				) AND b.profileID = ? ` + searchFilter

	err = db.Conn.Get(&count, stmt, profileID, profileID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get record's existence")
	}

	if count == 0 {
		return util.SetPaginationResponse(nil, count, 1, "Following boards not found"), nil
	}

	stmt = `SELECT
				b.boardTitle, 
				b.boardID,
				p.id as connectionProfileID,
				p.firstName, 
				p.lastName,
				p.screenName 
			FROM` +
		"`sidekiq-dev`.AccountProfile as p" + `
				INNER JOIN ` +
		"`sidekiq-dev`.BoardsFollowed as b ON b.ownerID = p.id " + boardFilter + `
			WHERE
				p.id IN 
				(
				SELECT
					ownerID
				FROM` +
		"`sidekiq-dev`.BoardsFollowed" + `
				WHERE
					profileID = ?
				) AND b.profileID = ? ` + searchFilter + ` LIMIT ` + limit + ` OFFSET ` + fmt.Sprintf("%v", offset)

	err = db.Conn.Select(&connections, stmt, profileID, profileID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch following boards")
	}

	for i := range connections {
		if len(res) == 0 {
			var temp model.FollowingInfo
			temp.BasicProfileInfo = connections[i].BasicProfileInfo
			temp.BoardDetails = append(temp.BoardDetails, connections[i].BoardInfo)
			res = append(res, temp)
		} else if len(res) > 0 {
			found := false
			for j := range res {
				if res[j].BasicProfileInfo.ID == connections[i].BasicProfileInfo.ID {
					res[j].BoardDetails = append(res[j].BoardDetails, connections[i].BoardInfo)
					found = true
					break
				}
			}
			if !found {
				var temp model.FollowingInfo
				temp.BasicProfileInfo = connections[i].BasicProfileInfo
				temp.BoardDetails = append(temp.BoardDetails, connections[i].BoardInfo)
				res = append(res, temp)
			}
		}
	}

	for i := range res {
		res[i].BasicProfileInfo.Photo, err = helper.GetProfileImage(db, storageService, 0, res[i].ID)
		if err != nil {
			fmt.Println("error in fetching profile photo")
		}

		res[i].BasicProfileInfo.Thumbs, err = helper.GetProfileImageThumb(db, storageService, 0, res[i].ID)
		if err != nil {
			fmt.Println("error in fetching profile thumb photo")
		}
	}

	return util.SetPaginationResponse(res, count, 1, "Following boards fetched successfully"), nil
}

func fetchConnectionRequests(db *mongodatabase.DBConfig, mysql *database.Database, storageService storage.Service, profileID int, limit, page string) (map[string]interface{}, error) {
	profileIDStr := strconv.Itoa(profileID)
	dbConn, err := db.New(consts.Request)
	if err != nil {
		return nil, err
	}
	collection, client := dbConn.Collection, dbConn.Client
	defer client.Disconnect(context.TODO())

	findOptions := options.Find()
	findOptions.SetSort(bson.M{"createDate": -1})

	var connReq []model.ConnectionRequest
	filter := bson.M{"profileID": profileIDStr}
	cursor, err := collection.Find(context.TODO(), filter, findOptions)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find connections")
	}
	err = cursor.All(context.TODO(), &connReq)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find profile's connections.")
	}
	ll, err := strconv.Atoi(limit)
	if err != nil {
		return nil, errors.Wrap(err, "string to int conversion limit")
	}

	pp, err := strconv.Atoi(page)
	if err != nil {
		return nil, errors.Wrap(err, "string to int conversion page")
	}

	var dd []interface{}
	for _, value := range connReq {
		dd = append(dd, value)
	}

	data := util.PaginateFromArray(dd, pp, ll)

	// Fetch QR from wasabi
	for i := range data {
		// Perform operations on each item as an interface
		if val, ok := data[i].(model.ConnectionRequest); ok {
			profileIDInt, err := strconv.Atoi(val.ProfileID)
			if err != nil {
				return nil, errors.Wrap(err, "unable to convert to int")
			}
			cp, err := helper.GetConciseProfile(mysql, profileIDInt, storageService)
			if err != nil {
				return nil, errors.Wrap(err, "unable to fetch concise profile")
			}
			awsKey := util.GetKeyForProfileQR(cp.UserID, profileIDInt)
			fileName := fmt.Sprintf("%s_%s_%s.png", cp.FirstName, cp.LastName, val.Code)
			fileData, err := storageService.GetUserFile(awsKey, fileName)
			if err != nil {
				return nil, errors.Wrap(err, "unable to fetch QR from wasabi")
			}
			val.QR = fileData.Filename
			data[i] = val
		}
	}
	return util.SetPaginationResponse(data, len(connReq), 1, "Connection requests fetched successfully."), nil
}

func fetchArchivedConnections(db *mongodatabase.DBConfig, mysql *database.Database, storageService storage.Service, profileID int, limit, page string, searchParameter ...string) (map[string]interface{}, error) {
	profileIDStr := strconv.Itoa(profileID)
	dbConn, err := db.New("Connection")
	if err != nil {
		return nil, err
	}

	connCollection, connClient := dbConn.Collection, dbConn.Client
	defer connClient.Disconnect(context.TODO())

	pageInt, err := strconv.Atoi(page)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert string to int")
	}
	limitInt, err := strconv.Atoi(limit)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert string to int")
	}

	offset := limitInt * (pageInt - 1)

	searchPattern := fmt.Sprintf(".*%s.*", searchParameter[0])

	countPipeline := mongo.Pipeline{
		bson.D{
			{Key: "$match", Value: bson.D{
				{Key: "$and", Value: bson.A{
					bson.D{{Key: "profileID", Value: profileIDStr}},
					bson.M{"isArchived": true},
				}},
			}},
		},
		bson.D{
			{Key: "$project", Value: bson.D{
				{Key: "fullName", Value: bson.D{
					{Key: "$concat", Value: bson.A{"$firstName", " ", "$lastName"}},
				}},
			}},
		},
		bson.D{
			{Key: "$match", Value: bson.D{
				{Key: "$or", Value: bson.A{
					bson.D{{Key: "firstName", Value: primitive.Regex{Pattern: searchPattern, Options: "i"}}},
					bson.D{{Key: "lastName", Value: primitive.Regex{Pattern: searchPattern, Options: "i"}}},
					bson.D{{Key: "fullName", Value: primitive.Regex{Pattern: searchPattern, Options: "i"}}},
					bson.D{{Key: "screenName", Value: primitive.Regex{Pattern: searchPattern, Options: "i"}}},
				}},
			}},
		},
		bson.D{
			{Key: "$count", Value: "count"},
		},
	}

	pipeline := mongo.Pipeline{
		bson.D{
			{Key: "$match", Value: bson.M{
				"profileID":  profileIDStr,
				"isArchived": true,
			}},
		},
		bson.D{
			{Key: "$project", Value: bson.M{
				"fullName": bson.M{
					"$concat": bson.A{"$firstName", " ", "$lastName"},
				},
				"firstName":    1,
				"connectionID": 1,
				"lastName":     1,
				"screenName":   1,
			}},
		},
		bson.D{
			{Key: "$match", Value: bson.M{
				"$or": bson.A{
					bson.M{"firstName": primitive.Regex{Pattern: searchPattern, Options: "i"}},
					bson.M{"lastName": primitive.Regex{Pattern: searchPattern, Options: "i"}},
					bson.M{"fullName": primitive.Regex{Pattern: searchPattern, Options: "i"}},
					bson.M{"screenName": primitive.Regex{Pattern: searchPattern, Options: "i"}},
				},
			}},
		},
		bson.D{
			{Key: "$skip", Value: offset},
		},
		bson.D{
			{Key: "$limit", Value: limitInt},
		},
	}

	var result struct {
		Count int `bson:"count"`
	}

	cursor, err := connCollection.Aggregate(context.TODO(), countPipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	if cursor.Next(context.TODO()) {
		err = cursor.Decode(&result)
		if err != nil {
			return nil, err
		}
	}

	cursor, err = connCollection.Aggregate(context.Background(), pipeline)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find archived connections")
	}

	connections := []model.BoardMemberRole{}

	err = cursor.All(context.TODO(), &connections)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find archived connections.")
	}

	for i := range connections {
		profileIDInt, _ := strconv.Atoi(connections[i].ProfileID)
		connections[i].Photo, err = helper.GetProfileImage(mysql, storageService, 0, profileIDInt)
		if err != nil {
			fmt.Println("error in fetching profile photo")
		}

		connections[i].Thumbs, err = helper.GetProfileImageThumb(mysql, storageService, 0, profileIDInt)
		if err != nil {
			fmt.Println("error in fetching profile thumbs photo")
		}
	}

	return util.SetPaginationResponse(connections, int(result.Count), 1, "Archived fetched successfully"), nil
}

func fetchBlockedConnections(db *mongodatabase.DBConfig, mysql *database.Database, storageService storage.Service, profileID int, limit, page string, searchParameter ...string) (map[string]interface{}, error) {
	profileIDStr := strconv.Itoa(profileID)
	dbConn, err := db.New("Connection")
	if err != nil {
		return nil, err
	}

	connCollection, connClient := dbConn.Collection, dbConn.Client
	defer connClient.Disconnect(context.TODO())

	pageInt, err := strconv.Atoi(page)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert string to int")
	}
	limitInt, err := strconv.Atoi(limit)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert string to int")
	}

	offset := limitInt * (pageInt - 1)

	searchPattern := fmt.Sprintf(".*%s.*", searchParameter[0])

	countPipeline := mongo.Pipeline{
		bson.D{
			{Key: "$match", Value: bson.D{
				{Key: "$and", Value: bson.A{
					bson.D{{Key: "profileID", Value: profileIDStr}},
					bson.M{"isBlocked": true},
				}},
			}},
		},
		bson.D{
			{Key: "$project", Value: bson.D{
				{Key: "fullName", Value: bson.D{
					{Key: "$concat", Value: bson.A{"$firstName", " ", "$lastName"}},
				}},
			}},
		},
		bson.D{
			{Key: "$match", Value: bson.D{
				{Key: "$or", Value: bson.A{
					bson.D{{Key: "firstName", Value: primitive.Regex{Pattern: searchPattern, Options: "i"}}},
					bson.D{{Key: "lastName", Value: primitive.Regex{Pattern: searchPattern, Options: "i"}}},
					bson.D{{Key: "fullName", Value: primitive.Regex{Pattern: searchPattern, Options: "i"}}},
					bson.D{{Key: "screenName", Value: primitive.Regex{Pattern: searchPattern, Options: "i"}}},
				}},
			}},
		},
		bson.D{
			{Key: "$count", Value: "count"},
		},
	}

	pipeline := mongo.Pipeline{
		bson.D{
			{Key: "$match", Value: bson.M{
				"profileID": profileIDStr,
				"isBlocked": true,
			}},
		},
		bson.D{
			{Key: "$project", Value: bson.M{
				"fullName": bson.M{
					"$concat": bson.A{"$firstName", " ", "$lastName"},
				},
				"firstName":    1,
				"connectionID": 1,
				"lastName":     1,
				"screenName":   1,
			}},
		},
		bson.D{
			{Key: "$match", Value: bson.M{
				"$or": bson.A{
					bson.M{"firstName": primitive.Regex{Pattern: searchPattern, Options: "i"}},
					bson.M{"lastName": primitive.Regex{Pattern: searchPattern, Options: "i"}},
					bson.M{"fullName": primitive.Regex{Pattern: searchPattern, Options: "i"}},
					bson.M{"screenName": primitive.Regex{Pattern: searchPattern, Options: "i"}},
				},
			}},
		},
		bson.D{
			{Key: "$skip", Value: offset},
		},
		bson.D{
			{Key: "$limit", Value: limitInt},
		},
	}

	var result struct {
		Count int `bson:"count"`
	}

	cursor, err := connCollection.Aggregate(context.TODO(), countPipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	if cursor.Next(context.TODO()) {
		err = cursor.Decode(&result)
		if err != nil {
			return nil, err
		}
	}

	cursor, err = connCollection.Aggregate(context.Background(), pipeline)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find archived connections")
	}

	connections := []model.BoardMemberRole{}

	err = cursor.All(context.TODO(), &connections)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find archived connections.")
	}

	for i := range connections {
		profileIDInt, _ := strconv.Atoi(connections[i].ProfileID)
		connections[i].Photo, err = helper.GetProfileImage(mysql, storageService, 0, profileIDInt)
		if err != nil {
			fmt.Println("error in fetching profile photo")
		}

		connections[i].Thumbs, err = helper.GetProfileImageThumb(mysql, storageService, 0, profileIDInt)
		if err != nil {
			fmt.Println("error in fetching profile thumb photo")
		}
	}

	return util.SetPaginationResponse(connections, int(result.Count), 1, "Profile blocked connections fetched successfully"), nil
}

func fetchBoards(db *database.Database, profileID int, sectionType string) (map[string]interface{}, error) {
	var boards []struct {
		BoardTitle string `json:"boardTitle" db:"boardTitle"`
		BoardID    string `json:"boardID" db:"boardID"`
	}
	var boardStmt string
	var err error

	if sectionType == "following" {
		boardStmt = "SELECT DISTINCT boardTitle, boardID FROM `sidekiq-dev`.BoardsFollowed WHERE profileID = ?"
	} else if sectionType == "followed" {
		boardStmt = "SELECT DISTINCT boardTitle, boardID FROM `sidekiq-dev`.BoardsFollowed WHERE ownerID = ?"
	} else {
		return util.SetResponse(nil, 0, "Invalid parameter in requested URL"), nil
	}

	err = db.Conn.Select(&boards, boardStmt, profileID)
	if err != nil {
		return nil, err
	}

	return util.SetResponse(boards, 1, cases.Title(language.English).String(strings.ToLower(sectionType))+" boards fetched successfully"), nil
}

func listAllOpenProfiles(db *database.Database) (map[string]interface{}, error) {
	stmt := "select id, IFNULL(photo, '') as photo, firstName, lastName, screenName FROM `sidekiq-dev`.AccountProfile WHERE visibility = ? AND searchable = ? AND isActive = ?"
	var profiles []model.ConciseProfile
	err := db.Conn.Select(&profiles, stmt, "Public", 1, 1)
	if err != nil {
		return nil, err
	}
	return util.SetResponse(profiles, 1, "Candidates for co-managers fetched successfully."), nil
}

func moveConnection(db *mongodatabase.DBConfig, payload map[string]interface{}, profileID int) (map[string]interface{}, error) {
	var connectionIDs []string
	var action bool
	var statusType string

	for _, record := range payload {
		switch reflect.TypeOf(record).Kind() {
		case reflect.Slice:
			s := reflect.ValueOf(record)

			for i := 0; i < s.Len(); i++ {
				var v interface{} = s.Index(i).Interface()
				connectionIDs = append(connectionIDs, v.(string))
			}
		}
		if rec, ok := record.(bool); ok {
			action = rec
		}
		if rec, ok := record.(string); ok {
			statusType = rec
		}
	}

	if len(connectionIDs) == 0 {
		return util.SetResponse(nil, 0, "ConnectionIDs not found"), nil
	}

	dbConn, err := db.New("Connection")
	if err != nil {
		return nil, err
	}
	connCollection, connClient := dbConn.Collection, dbConn.Client
	fmt.Println(connCollection)
	defer connClient.Disconnect(context.TODO())

	var filter primitive.M
	var update primitive.M

	profileIDStr := strconv.Itoa(profileID)

	for _, connectionProfileID := range connectionIDs {

		filter = bson.M{"$and": bson.A{
			bson.M{"profileID": profileIDStr},
			bson.M{"connectionID": connectionProfileID},
		}}

		switch statusType {
		case "archive":
			update = bson.M{"$set": bson.M{"isArchived": action}}
			if !action {
				// set isActive to true  and isBlocked = false
				update["$set"].(bson.M)["isActive"] = true
				update["$set"].(bson.M)["isBlocked"] = false
			} else {
				// set isActive and isBlocked to false
				update["$set"].(bson.M)["isActive"] = false
				update["$set"].(bson.M)["isBlocked"] = false
			}
		case "blocked":
			update = bson.M{"$set": bson.M{"isBlocked": action}}
			if !action {
				// set isActive to true  and isArchived = false
				update["$set"].(bson.M)["isActive"] = true
				update["$set"].(bson.M)["isArchived"] = false
			} else {
				// set isActive to false  and isArchived = false
				update["$set"].(bson.M)["isActive"] = false
				update["$set"].(bson.M)["isArchived"] = false
			}
		case "active":
			update = bson.M{"$set": bson.M{"isActive": action}}
			if !action {
				update["$set"].(bson.M)["isArchived"] = true
				update["$set"].(bson.M)["isBlocked"] = false
			} else {
				update["$set"].(bson.M)["isArchived"] = false
				update["$set"].(bson.M)["isBlocked"] = false
			}
		default:
			return util.SetResponse(nil, 0, "Request type invalid"), nil
		}

		fmt.Println("update: ", update)
		_, err = connCollection.UpdateOne(context.TODO(), filter, update)
		if err != nil {
			return util.SetResponse(nil, 0, "unable to perform update"), nil
		}
	}

	return util.SetResponse(nil, 1, "Connections moved successfully"), nil
}
