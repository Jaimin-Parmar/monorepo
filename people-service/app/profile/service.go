package profile

import (
	"people-service/app/config"
	"people-service/app/email"
	"people-service/app/storage"
	"people-service/cache"
	"people-service/database"
	"people-service/model"
	"people-service/mongodatabase"
)

// Service - defines Profile service
type Service interface {
	ValidateProfile(profileID int, accountID int) error
	GetProfilesByUserID(userID int) (map[string]interface{}, error)
	GetProfileCountByUserID(userID int) (map[string]interface{}, error)
	AddProfile(profile model.Profile, userID int) (map[string]interface{}, error)
	GetProfileInfo(profileID int) (map[string]interface{}, error)
	FetchTags(profileID int) (map[string]interface{}, error)
	EditProfile(profile model.Profile) (map[string]interface{}, error)
	UpdateProfileTagsNew(profileID string) error
	DeleteProfile(userID string, profileID []string) (map[string]interface{}, error)
}

type service struct {
	config         *config.Config
	dbMaster       *database.Database
	dbReplica      *database.Database
	mongodb        *mongodatabase.DBConfig
	cache          *cache.Cache
	emailService   email.Service
	storageService storage.Service
}

func NewService(repos *model.Repos, conf *config.Config) Service {
	return &service{
		config:         conf,
		mongodb:        repos.MongoDB,
		dbMaster:       repos.MasterDB,
		dbReplica:      repos.ReplicaDB,
		cache:          repos.Cache,
		emailService:   email.NewService(),
		storageService: storage.NewService(repos, conf),
	}
}

func (s *service) ValidateProfile(profileID int, accountID int) error {
	return ValidateProfileByAccountID(s.dbMaster, profileID, accountID)
}

func (s *service) GetProfilesByUserID(userID int) (map[string]interface{}, error) {
	return getProfilesByUserID(s.dbMaster, userID, s.storageService)
}

func (s *service) GetProfileCountByUserID(userID int) (map[string]interface{}, error) {
	return getProfileCountByUserID(s.dbMaster, userID)
}

func (s *service) AddProfile(profile model.Profile, userID int) (map[string]interface{}, error) {
	return addProfile(s.dbMaster, profile, userID)
}

func (s *service) GetProfileInfo(profileID int) (map[string]interface{}, error) {
	return getProfileInfo(s.dbMaster, profileID, s.storageService)
}

func (s *service) FetchTags(profileID int) (map[string]interface{}, error) {
	return fetchTags(s.dbMaster, profileID)
}

func (s *service) EditProfile(profile model.Profile) (map[string]interface{}, error) {
	return editProfile(s.dbMaster, profile)
}

func (s *service) UpdateProfileTagsNew(profileID string) error {
	return updateProfileTagsNew(s.mongodb, s.dbMaster, profileID)
}

func (s *service) DeleteProfile(userID string, profileIDs []string) (map[string]interface{}, error) {
	return deleteProfile(s.dbMaster, userID, profileIDs)
}
