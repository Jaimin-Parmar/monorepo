package account

import (
	"fmt"
	"people-service/cache"
	"people-service/database"
	"people-service/model"
	"people-service/util"

	"github.com/pkg/errors"
)

func fetchAccountForAuthByEmail(cache *cache.Cache, db *database.Database, email string) (*model.Account, error) {
	accountdata := &model.Account{}
	err := db.Conn.Get(accountdata, "SELECT id, email, password FROM `sidekiq-dev`.Account WHERE email = ?;", email)
	if err != nil {
		return nil, errors.New("incorrect email")
	}
	return accountdata, nil
}

func getCacheKey(accountID int) string {
	return fmt.Sprintf("user:%d", accountID)
}

func getAccountFromDB(db *database.Database, accountID int) (*model.Account, error) {
	stmt := "SELECT id, accountType, firstName, lastName, email, password, lastModifiedDate FROM `sidekiq-dev`.Account WHERE id = ?;"
	user := &model.Account{}
	err := db.Conn.Get(user, stmt, accountID)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func getAccountPermissions(db *database.Database, userID int) []*model.AccountPermimssion {
	stmt := "SELECT l.orgID, a.company, l.owner, l.apiAccess FROM `sidekiq-dev`.OrgProfile a LEFT JOIN `sidekiq-dev`.LinkUserToOrg l ON l.orgID = a.id WHERE l.userID = ?;"
	permissions := []*model.AccountPermimssion{}
	db.Conn.Select(&permissions, stmt, userID)
	return permissions
}

func fetchAccounts(db *database.Database) (map[string]interface{}, error) {
	response := make(map[string]interface{})
	accountTypes := []*model.AccountTypes{}
	stmt := "SELECT id, service, description, fee, profiles FROM `sidekiq-dev`.Services WHERE serviceType = 1;"
	err := db.Conn.Select(&accountTypes, stmt)
	fmt.Println(response)
	if err != nil {
		return nil, err
	}
	response = util.SetResponse(accountTypes, 1, "Request Successfully completed")
	return response, nil
}

func createAccount(db *database.Database, user model.AccountSignup) (map[string]interface{}, error) {
	var fetchstmt string
	var countUser, countAccount *int64
	resData := make(map[string]interface{})

	// Check if email already exists
	fetchstmt = "SELECT COUNT(*) AS COUNT FROM `sidekiq-dev`.Account WHERE email = ?"
	err := db.Conn.Get(&countUser, fetchstmt, user.Email)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch email count on account")
	}
	if (*countUser) != 0 {
		return util.SetResponse(nil, 0, "Account already exists with same email. Please use different email"), nil
	}

	// Check if phone number already exists
	fetchstmt = "SELECT COUNT(*) AS COUNT FROM `sidekiq-dev`.Account WHERE phone = ?"
	err = db.Conn.Get(&countUser, fetchstmt, user.Phone)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch phone count on Account")
	}
	if (*countUser) != 0 {
		return util.SetResponse(nil, 0, "Account already exists with same phone number. Please use different phone number"), nil
	}

	fetchstmt = "SELECT COUNT(*) AS COUNT FROM `sidekiq-dev`.AccountSignup WHERE phone = ? AND email = ?"
	err = db.Conn.Get(&countAccount, fetchstmt, user.Phone, user.Email)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch count on AccountSignup")
	}

	// Account exists in SignupUsers but not Users
	if (*countAccount) != 0 {
		u := &model.AccountSignup{}
		if user.Email != "" {
			fetchstmt = "SELECT id FROM `sidekiq-dev`.AccountSignup WHERE email = ? AND phone = ?"
			err := db.Conn.Get(u, fetchstmt, user.Email, user.Phone)
			if err != nil {
				return nil, errors.Wrap(err, "unable to fetch id")
			}
			resData["id"] = int64(u.ID)
			resData["email"] = user.Email
			resData["phone"] = user.Phone
		}
	} else {
		// Insert in SignupUsers
		stmt := "INSERT INTO `sidekiq-dev`.AccountSignup (email, phone) VALUES(:email, :phone)"
		r, err := db.Conn.NamedExec(stmt, user)
		if err != nil {
			return nil, errors.Wrap(err, "unable to insert user")
		}
		resData["id"], err = r.LastInsertId()
		if err != nil {
			return nil, err
		}
		resData["email"] = user.Email
		resData["phone"] = user.Phone
	}
	return util.SetResponse(resData, 1, "Account created successfully"), nil
}

// func getVerificationCode(db *database.Database, emailService email.Service, userID int, emailID string) (map[string]interface{}, error) {
// 	code, err := util.EncodeToString(6)
// 	if err != nil {
// 		return nil, errors.Wrap(err, "unable to generate code")
// 	}
// 	var user model.AccountSignup
// 	user.VerificationCode = code
// 	user.ID = userID
// 	stmt := "UPDATE `sidekiq-dev`.AccountSignup SET verificationCode = :verificationCode WHERE id = :id;"
// 	_, err = db.Conn.NamedExec(stmt, user)
// 	if err != nil {
// 		return nil, errors.Wrap(err, "unable to update verification code for user")
// 	}

// 	// send email
// 	email := model.Email{}
// 	email.Sender = "donotreply@otp.sidekiq.com" // don't hardcode, use default.yaml
// 	email.Receiver = emailID
// 	email.Subject = "Please Verify Your Email"
// 	email.HtmlBody = fmt.Sprintf(`<h3>Hey,<br>
// 		A sign in attempt requires further verification because we did not recognize your Email.
// 		To complete the sign in, enter the verification code on the given Email.<br><br>Verification Code: <b>%s</b></h3>`, code)
// 	email.TextBody = fmt.Sprintf("Hey. A sign in attempt requires further verification because we did not recognize your Email. To complete the sign in, enter the verification code on the given Email. Verification code: %s", code)
// 	err = emailService.SendEmail(email)
// 	if err != nil {
// 		return nil, errors.Wrap(err, "unable to send email for reset password")
// 	}

// 	// err = sendEmailService(userID, emailID, code, false)
// 	// if err != nil {
// 	// 	return nil, errors.Wrap(err, "unable to send verification code to user using SES")
// 	// }

// 	res := make(map[string]interface{})
// 	resData := make(map[string]interface{})
// 	resData["code"] = code
// 	resData["status"] = 1
// 	resData["message"] = "Verification code sent successfully"
// 	res["data"] = resData
// 	return resData, nil
// }

// func verifyLink(db *database.Database, emailService email.Service, token string) (map[string]interface{}, error) {
// 	response := make(map[string]interface{})
// 	user := []*model.Account{}

// 	// check if password already saved using the link
// 	// fetchstmt := "SELECT * FROM `sidekiq-dev`.Account WHERE resetToken = ?"
// 	fetchstmt := `SELECT
// 		id, accountType, createDate, lastModifiedDate, isActive,
// 		IFNULL(firstName, "") as firstName,
// 		IFNULL(lastName, "") as lastName,
// 		IFNULL(userName, "") as userName,
// 		IFNULL(email, "") as email,
// 		IFNULL(phone, "") as phone,
// 		IFNULL(recoveryEmail, "") as recoveryEmail,
// 		IFNULL(resetStatus, "") as resetStatus,
// 		IFNULL(resetTime, "") as resetTime

// 		FROM` + "`sidekiq-dev`.Account WHERE resetToken = ?"

// 	err := db.Conn.Select(&user, fetchstmt, token)
// 	if err != nil {
// 		response = util.SetResponse(nil, 0, "Error in processing request")
// 		return response, err
// 	}

// 	if (len(user)) == 1 {
// 		// password not saved using this link but link may have expired
// 		uniqueUser := user[0]
// 		resetStatus := uniqueUser.ResetStatus
// 		currentTime := time.Now()
// 		expireTime := currentTime.Add(-time.Minute * 10)

// 		resetTime, err := time.Parse("2006-01-02 15:04:05", string(uniqueUser.ResetTime))
// 		if err != nil {
// 			response = util.SetResponse(nil, 0, "Error in processing request")
// 			return response, err
// 		}

// 		// check if password not set using this token already and link is valid
// 		if resetStatus && !expireTime.After(resetTime) {
// 			// return response to frontend
// 			response = util.SetResponse(nil, 1, "Link Validation Completed successfully")
// 		} else if resetStatus {
// 			// generate uuid for sending email
// 			uuid := uuid.New().String()

// 			// db store
// 			var payload struct {
// 				Email string `json:"email" db:"email"`
// 				UUID  string `json:"resetToken" db:"resetToken"`
// 			}
// 			payload.UUID = uuid
// 			payload.Email = uniqueUser.Email
// 			stmt := "UPDATE `sidekiq-dev`.Account SET resetToken=:resetToken, resetTime = now(), resetStatus = true WHERE email = :email;"
// 			_, err := db.Conn.NamedExec(stmt, payload)
// 			if err != nil {
// 				response = util.SetResponse(nil, 0, "Error in processing request")
// 				return response, err
// 			}

// 			// create reset link
// 			resetPageLink := "https://staging.sidekiq.com/reset-password/" // from frontend
// 			link := resetPageLink + uuid

// 			email := model.Email{}
// 			email.Receiver = uniqueUser.Email
// 			email.Header = "Sidekiq: Reset password link verfication"
// 			email.Subject = "Link to reset password"
// 			email.HtmlBody = fmt.Sprintf(`<h3>Hey,
// 				<br>Please <a href="%s">click here</a> to reset your password.
// 				The link will automatically expire after 10 minutes</h3>`, link)
// 			email.TextBody = fmt.Sprintf(`Hey. Please <a href="%s">click here</a> to reset your password.
// 				The link will automatically expire after 10 minutes`, link)
// 			err = emailService.SendEmail(email)
// 			if err != nil {
// 				response = util.SetResponse(nil, 0, "Unable to send reset link on your email")
// 				return response, err
// 			}
// 			response = util.SetResponse(nil, 0, "This link has expired. A new link has been sent on your email")

// 			// send reset email
// 			// err = sendEmailService(-1, uniqueUser.Email, link, true)
// 			// if err != nil {
// 			// 	response = util.SetResponse(nil, 0, "Unable to send reset link on your email")
// 			// 	return response, err
// 			// }
// 			// response = util.SetResponse(nil, 0, "This link has expired. A new link has been sent on your email")
// 		}
// 	} else {
// 		response = util.SetResponse(nil, 0, "You can't reset password with this link.")
// 	}
// 	return response, nil
// }

// func forgotPassword(db *database.Database, emailService email.Service, recipientEmail string) (map[string]interface{}, error) {
// 	/* Flow -
// 	   1. Check if account exists in DB from email
// 	   2. Generate uuid.
// 	   3. Store uuid in DB and attach in reset link.
// 	   4. Send reset link on recipient email.
// 	   5. Return success response with status 1 and appropriate message
// 	*/

// 	response := make(map[string]interface{})
// 	var countUser *int64

// 	// check if account exists
// 	if recipientEmail != "" {
// 		fetchstmt := "SELECT COUNT(*) AS COUNT FROM `sidekiq-dev`.Account WHERE email = ?"
// 		err := db.Conn.Get(&countUser, fetchstmt, recipientEmail)
// 		if err != nil {
// 			response = util.SetResponse(nil, 0, "Error in processing request")
// 			return response, err
// 		}
// 	} else {
// 		response = util.SetResponse(nil, 0, "Email missing")
// 	}

// 	if (*countUser) != 0 {
// 		// generate uuid for sending email
// 		uuid := uuid.New().String()

// 		// db store
// 		var payload struct {
// 			Email string `json:"email" db:"email"`
// 			UUID  string `json:"resetToken" db:"resetToken"`
// 		}
// 		payload.UUID = uuid
// 		payload.Email = recipientEmail
// 		stmt := "UPDATE `sidekiq-dev`.Account SET resetToken=:resetToken, resetTime = now(), resetStatus = true WHERE email = :email;"
// 		_, err := db.Conn.NamedExec(stmt, payload)
// 		if err != nil {
// 			response = util.SetResponse(nil, 0, "Error in processing request")
// 			return response, err
// 		}

// 		// create reset link based on env value
// 		resetPageLink := "https://staging.sidekiq.com/reset-password/" // from frontend
// 		// resetPageLink := "https://sidekiq.com/reset-password/" // from frontend
// 		link := resetPageLink + uuid

// 		email := model.Email{}
// 		email.Sender = "donotreply@otp.sidekiq.com"
// 		email.Receiver = recipientEmail
// 		email.Header = "Sidekiq: Forgot password"
// 		email.Subject = "Forgot Password"
// 		email.HtmlBody = fmt.Sprintf(`<h3>Hey,
// 			<br>Please <a href="%s">click here</a> to reset your password.
// 			The link will automatically expire after 10 minutes</h3>`, link)
// 		email.TextBody = fmt.Sprintf(`Hey. Please <a href="%s">click here</a> to reset your password. The link will automatically expire after 10 minutes`, link)
// 		err = emailService.SendEmail(email)
// 		if err != nil {
// 			response = util.SetResponse(nil, 0, "Unable to send reset link on your email")
// 			return response, err
// 		}

// 		// send reset email
// 		// err = sendEmailService(-1, recipientEmail, link, true)
// 		// if err != nil {
// 		// 	response = util.SetResponse(nil, 0, "Unable to send reset link on your email")
// 		// 	return response, err
// 		// }

// 		// return response to frontend
// 		response = util.SetResponse(nil, 1, "Password reset link sent successfully")
// 	} else {
// 		response = util.SetResponse(nil, 0, "Account does not exist for this email")
// 	}
// 	return response, nil
// }

// func resetPassword(db *database.Database, emailService email.Service, payload *model.ResetPassword) (map[string]interface{}, error) {
// 	/* Flow -
// 	   1. Check if link expired or password already saved once if not save password based on token.
// 	   2. Set resetStatus to false once token is saved.
// 	   3. Return success response with status 1 and appropriate message
// 	*/

// 	response := make(map[string]interface{})
// 	user := []*model.Account{}

// 	// check if password already saved once
// 	fetchstmt := "SELECT * FROM `sidekiq-dev`.Account WHERE resetToken = ?"
// 	err := db.Conn.Select(&user, fetchstmt, payload.ResetToken)
// 	if err != nil {
// 		response = util.SetResponse(nil, 0, "Error in processing request")
// 		return response, err
// 	}

// 	if (len(user)) == 1 {
// 		uniqueUser := user[0]
// 		resetStatus := uniqueUser.ResetStatus
// 		currentTime := time.Now()
// 		expireTime := currentTime.Add(-time.Minute * 10)

// 		resetTime, err := time.Parse(time.RFC3339, string(uniqueUser.ResetTime))
// 		if err != nil {
// 			response = util.SetResponse(nil, 0, "Error in processing request")
// 			return response, err
// 		}
// 		// check reset status (if password is updated using this token) or else it is expired.
// 		if !resetStatus || expireTime.After(resetTime) {
// 			// generate uuid for sending email
// 			uuid := uuid.New().String()

// 			// db store
// 			var tokenStructure struct {
// 				Email string `json:"email" db:"email"`
// 				UUID  string `json:"resetToken" db:"resetToken"`
// 			}
// 			tokenStructure.UUID = uuid
// 			tokenStructure.Email = uniqueUser.Email
// 			stmt := "UPDATE `sidekiq-dev`.Account SET resetToken=:resetToken, resetTime = now(), resetStatus = true WHERE email = :email;"
// 			_, err := db.Conn.NamedExec(stmt, tokenStructure)
// 			if err != nil {
// 				response = util.SetResponse(nil, 0, "Error in processing request")
// 				return response, err
// 			}

// 			fmt.Println("called: ", 668)

// 			// create reset link
// 			resetPageLink := "http://35.170.215.50/reset-password/" // from frontend
// 			link := resetPageLink + uuid

// 			email := model.Email{}
// 			email.Sender = "donotreply@otp.sidekiq.com"
// 			email.Receiver = payload.ResetToken
// 			email.Header = "Sidekiq: reset password"
// 			email.Subject = "Reset Password"
// 			email.HtmlBody = fmt.Sprintf(`Hey<br>Please <a href="%s">click here</a> to reset your password. The link will automatically expire after 10 minutes`, link)
// 			email.TextBody = fmt.Sprintf(`Hey. Please <a href="%s">click here</a> to reset your password. The link will automatically expire after 10 minutes`, link)
// 			err = emailService.SendEmail(email)
// 			if err != nil {
// 				response = util.SetResponse(nil, 0, "Unable to send reset link on your email")
// 				return response, err
// 			}

// 			// send reset email
// 			// err = sendEmailService(-1, payload.ResetToken, link, true)
// 			// if err != nil {
// 			// 	response = util.SetResponse(nil, 0, "Unable to send reset link on your email")
// 			// 	return response, err
// 			// }
// 			response = util.SetResponse(nil, 0, "This link has expired. A new link has sent on the given email ID")
// 		} else {
// 			// save new password in DB based on resetToken and also set resetStatus to false since password should be set only once from this link.
// 			stmt := "UPDATE `sidekiq-dev`.Account SET password=:password, resetStatus = false, resetToken = '' WHERE resetToken = :resetToken AND resetStatus = true;"
// 			_, err = db.Conn.NamedExec(stmt, payload)
// 			if err != nil {
// 				fmt.Println("Error in update password query")
// 				response = util.SetResponse(nil, 0, "Error in processing request")
// 				return response, err
// 			}
// 			fmt.Println("New Password set successfully")

// 			// return response to frontend
// 			response = util.SetResponse(nil, 1, "Password reset successful.")
// 		}
// 	} else {
// 		response = util.SetResponse(nil, 0, "Invalid Link. You can't reset password with this link.")
// 	}
// 	return response, nil
// }

// func setAccountType(db *database.Database, storageService storage.Service, payload *model.SetAccountType) (map[string]interface{}, error) {

// 	var countUser *int64

// 	fetchstmt := "SELECT COUNT(*) AS COUNT FROM `sidekiq-dev`.Account WHERE id = ?"
// 	err := db.Conn.Get(&countUser, fetchstmt, payload.AccountId)
// 	if err != nil {
// 		return util.SetResponse(nil, 0, "Error in processing request"), err
// 	}

// 	if (*countUser) != 0 {
// 		stmt := "UPDATE `sidekiq-dev`.Account SET accountType=:accountType WHERE id = :id"
// 		_, err := db.Conn.NamedExec(stmt, payload)
// 		if err != nil {
// 			return util.SetResponse(nil, 0, "Error in processing request"), errors.Wrap(err, "Error in updating accountType in DB")
// 		}

// 		stmt = `SELECT
// 			id, accountType, createDate, lastModifiedDate, isActive,
// 			IFNULL(firstName, "") as firstName,
// 			IFNULL(lastName, "") as lastName,
// 			IFNULL(userName, "") as userName,
// 			IFNULL(email, "") as email,
// 			IFNULL(phone, "") as phone,
// 			IFNULL(recoveryEmail, "") as recoveryEmail

// 			FROM` + "`sidekiq-dev`.Account WHERE id = ?"

// 		user := model.Account{}
// 		err = db.Conn.Get(&user, stmt, payload.AccountId)
// 		if err != nil {
// 			return util.SetResponse(nil, 0, "Error in processing request"), errors.Wrap(err, "unable to fetch user info")
// 		}

// 		stmt = `SELECT id, service, description, fee, profiles FROM` + "`sidekiq-dev`.Services WHERE id = ?"

// 		accountTypedetails := model.AccountTypes{}
// 		err = db.Conn.Get(&accountTypedetails, stmt, user.AccountType)
// 		if err != nil {
// 			return util.SetResponse(nil, 0, "Error in processing request"), errors.Wrap(err, "unable to fetch account type")
// 		}

// 		user.Photo, err = getAccountImage(db, storageService, user.ID, 0)
// 		if err != nil {
// 			user.Photo = ""
// 		}

// 		user.Thumbs, err = getAccountImageThumb(db, storageService, user.ID)
// 		if err != nil {
// 			user.Thumbs = model.Thumbnails{}
// 		}

// 		res := make(map[string]interface{})
// 		res["accountDetails"] = user
// 		res["accountTypeDetails"] = accountTypedetails
// 		return util.SetResponse(res, 1, "Account type set successfully"), nil
// 	}

// 	return util.SetResponse(nil, 0, "User associated with the id not found"), nil
// }