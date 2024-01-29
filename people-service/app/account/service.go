package account

import (
	"encoding/json"
	"errors"

	"people-service/app/config"
	"people-service/app/email"

	"people-service/cache"

	"people-service/database"

	"people-service/model"
)

// Service defines service for operating on Accounts
type Service interface {
	AuthAccount(creds *model.Credentials) (*model.Account, error)
	FetchAccounts() (map[string]interface{}, error)
	FetchAccount(id int, skipCache bool) (*model.Account, error)
	FetchContacts(accountID int) ([]*model.Contact, error)
	CreateAccount(account model.AccountSignup) (map[string]interface{}, error)
	FetchCachedAccount(id int) (*model.Account, error)
	GetVerificationCode(accountID int, emailID string) (map[string]interface{}, error)
	VerifyLink(token string) (map[string]interface{}, error)
	// ForgotPassword(userEmail string) (map[string]interface{}, error)
	// ResetPassword(payload *model.ResetPassword) (map[string]interface{}, error)
	// SetAccountType(payload *model.SetAccountType) (map[string]interface{}, error)
}

type service struct {
	config       *config.Config
	dbMaster     *database.Database
	dbReplica    *database.Database
	cache        *cache.Cache
	emailService email.Service
}

// NewService create new AccountService
func NewService(repos *model.Repos, conf *config.Config) Service {
	svc := &service{
		config:       conf,
		dbMaster:     repos.MasterDB,
		dbReplica:    repos.ReplicaDB,
		cache:        repos.Cache,
		emailService: email.NewService(),
	}
	return svc
}

// AuthAccount - fetches account and verifies their password
func (s *service) AuthAccount(creds *model.Credentials) (*model.Account, error) {
	accountData, err := fetchAccountForAuthByEmail(s.cache, s.dbMaster, creds.Email)
	if err != nil {
		return nil, err
	}
	if accountData == nil {
		return nil, nil
	}
	if accountData.Password != creds.Password {
		return nil, errors.New("incorrect password")
	}
	newAccountObj, err := s.FetchAccount(accountData.ID, true)
	if err != nil {
		return nil, err
	}
	return newAccountObj, nil
}

func (s *service) FetchAccount(id int, skipCache bool) (*model.Account, error) {
	key := getCacheKey(id)
	if !skipCache {
		cachedAccount, err := s.FetchCachedAccount(id)
		if err != nil {
			return nil, err
		}
		if cachedAccount == nil {
			return nil, errors.New("cachedAccount empty")
		}
		return cachedAccount, nil
	}
	accountData, err := getAccountFromDB(s.dbMaster, id)
	if err != nil {
		return nil, err
	}
	accountData.Accounts = getAccountPermissions(s.dbMaster, id)
	err = s.cache.SetValue(key, accountData.ToJSON())
	if err != nil {
		return nil, err
	}
	s.cache.ExpireKey(key, cache.Expire18HR)
	return accountData, nil
}

func (s *service) FetchCachedAccount(id int) (*model.Account, error) {
	key := getCacheKey(id)
	val, err := s.cache.GetValue(key)
	if err != nil {
		return nil, err
	}
	var accountData *model.Account
	json.Unmarshal([]byte(val), &accountData)
	return accountData, nil
}

func (s *service) FetchAccounts() (map[string]interface{}, error) {
	return fetchAccounts(s.dbMaster)
}

func (s *service) FetchContacts(userID int) ([]*model.Contact, error) {
	return getContacts(s.dbMaster, userID)
}

func (s *service) CreateAccount(account model.AccountSignup) (map[string]interface{}, error) {
	return createAccount(s.dbMaster, account)
}

func (s *service) GetVerificationCode(accountID int, emailID string) (map[string]interface{}, error) {
	return getVerificationCode(s.dbMaster, s.emailService, accountID, emailID)
}

func (s *service) VerifyLink(token string) (map[string]interface{}, error) {
	return verifyLink(s.dbMaster, s.emailService, token)
}

// func (s *service) ForgotPassword(userEmail string) (map[string]interface{}, error) {
// 	return forgotPassword(s.dbMaster, s.emailService, userEmail)
// }

// func (s *service) ResetPassword(payload *model.ResetPassword) (map[string]interface{}, error) {
// 	return resetPassword(s.dbMaster, s.emailService, payload)
// }

// func (s *service) SetAccountType(payload *model.SetAccountType) (map[string]interface{}, error) {
// 	return setAccountType(s.dbMaster, s.storageService, payload)
// }
