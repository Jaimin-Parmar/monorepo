package profile

import (
	"people-service/app/config"
	"people-service/app/email"
	"people-service/cache"
	"people-service/database"
	"people-service/model"
	"people-service/mongodatabase"
)

// Service - defines Profile service
type Service interface {
	ValidateProfile(profileID int, accountID int) error
}

type service struct {
	config       *config.Config
	dbMaster     *database.Database
	dbReplica    *database.Database
	mongodb      *mongodatabase.DBConfig
	cache        *cache.Cache
	emailService email.Service
}

func NewService(repos *model.Repos, conf *config.Config) Service {
	return &service{
		config:       conf,
		mongodb:      repos.MongoDB,
		dbMaster:     repos.MasterDB,
		dbReplica:    repos.ReplicaDB,
		cache:        repos.Cache,
		emailService: email.NewService(),
	}
}

func (s *service) ValidateProfile(profileID int, accountID int) error {
	return ValidateProfileByAccountID(s.dbMaster, profileID, accountID)
}
