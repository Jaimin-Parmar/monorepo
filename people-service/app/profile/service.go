package profile

import (
	"people-service/app/config"
	"people-service/app/email"
	"people-service/app/storage"
	"people-service/cache"
	"people-service/database"
	"people-service/model"
	"people-service/mongodatabase"
	"people-service/util"
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
	GenerateCode(userID, CallerprofileID int, payload model.ConnectionRequest) (map[string]interface{}, error)
	DeleteCode(userID int, codes map[string]interface{}) (map[string]interface{}, error)
	UpdateProfileSettings(profile model.UpdateProfileSettings, userID int) (map[string]interface{}, error)
	UpdateShareableSettings(profileID int, shareableSettings model.ShareableSettings) (map[string]interface{}, error)
	SendCoManagerRequest(profileID int, connReq map[string]interface{}) (map[string]interface{}, error)
	AcceptCoManagerRequest(userID, profileID int, code string) (map[string]interface{}, error)
	GetProfilesWithInfoByUserID(userID int) (map[string]interface{}, error)
	FetchProfilesWithCoManager(profileID int, profiles []model.Profile, search, page, limit string) (map[string]interface{}, error)
	FetchExternalProfiles(userID int, search, page, limit string) (map[string]interface{}, error)
	GetPeopleInfo(profileID, requestType int, limit, page string, searchParameter ...string) (map[string]interface{}, error)
	FetchBoards(profileID int, sectionType string) (map[string]interface{}, error)
	ListAllOpenProfiles() (map[string]interface{}, error)
	MoveConnection(payload map[string]interface{}, profileID int) (map[string]interface{}, error)
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

func (s *service) GenerateCode(userID, CallerprofileID int, payload model.ConnectionRequest) (map[string]interface{}, error) {
	return generateCode(s.mongodb, s.dbMaster, s.storageService, userID, CallerprofileID, payload)
}

func (s *service) DeleteCode(userID int, codes map[string]interface{}) (map[string]interface{}, error) {
	return deleteCode(s.mongodb, s.dbMaster, s.storageService, userID, codes)
}

func (s *service) UpdateProfileSettings(profile model.UpdateProfileSettings, userID int) (map[string]interface{}, error) {
	return updateProfileSettings(s.dbMaster, profile, userID)
}

func (s *service) UpdateShareableSettings(profileID int, shareableSettings model.ShareableSettings) (map[string]interface{}, error) {
	return updateShareableSettings(s.dbMaster, s.mongodb, profileID, shareableSettings)
}

func (s *service) SendCoManagerRequest(profileID int, connReq map[string]interface{}) (map[string]interface{}, error) {
	return sendCoManagerRequest(s.dbMaster, s.mongodb, s.emailService, profileID, connReq)
}

func (s *service) AcceptCoManagerRequest(userID, profileID int, code string) (map[string]interface{}, error) {
	return acceptCoManagerRequest(s.dbMaster, s.mongodb, s.storageService, userID, profileID, code)
}

func (s *service) GetProfilesWithInfoByUserID(userID int) (map[string]interface{}, error) {
	return getProfilesWithInfoByUserID(s.storageService, s.dbMaster, userID)
}

func (s *service) FetchProfilesWithCoManager(profileID int, profiles []model.Profile, search, page, limit string) (map[string]interface{}, error) {
	return fetchProfilesWithCoManager(s.storageService, s.dbMaster, s.mongodb, profileID, profiles, search, page, limit)
}

func (s *service) FetchExternalProfiles(userID int, search, page, limit string) (map[string]interface{}, error) {
	return fetchExternalProfiles(s.storageService, s.dbMaster, userID, search, page, limit)
}

func (s *service) GetPeopleInfo(profileID, requestType int, limit, page string, searchParameter ...string) (map[string]interface{}, error) {
	switch requestType {
	case 1:
		return fetchProfileConnections(s.mongodb, s.dbMaster, s.storageService, profileID, limit, page, searchParameter[0])
	case 2:
		return fetchBoardFollowers(s.dbMaster, s.storageService, profileID, limit, page, searchParameter[0], searchParameter[1])
	case 3:
		return fetchFollowingBoards(s.dbMaster, s.storageService, profileID, limit, page, searchParameter[0], searchParameter[1])
	case 4:
		return fetchConnectionRequests(s.mongodb, s.dbMaster, s.storageService, profileID, limit, page)
	case 5:
		return fetchArchivedConnections(s.mongodb, s.dbMaster, s.storageService, profileID, limit, page, searchParameter[0])
	case 6:
		return fetchBlockedConnections(s.mongodb, s.dbMaster, s.storageService, profileID, limit, page, searchParameter[0])
	}
	return util.SetResponse(nil, 0, "Request type invalid"), nil
}

func (s *service) FetchBoards(profileID int, sectionType string) (map[string]interface{}, error) {
	return fetchBoards(s.dbMaster, profileID, sectionType)
}

func (s *service) ListAllOpenProfiles() (map[string]interface{}, error) {
	return listAllOpenProfiles(s.dbMaster)
}

func (s *service) MoveConnection(payload map[string]interface{}, profileID int) (map[string]interface{}, error) {
	return moveConnection(s.mongodb, payload, profileID)
}
