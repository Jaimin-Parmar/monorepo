package storage

import (
	"people-service/app/config"
	"people-service/cache"
	"people-service/database"
	"people-service/model"
	"people-service/mongodatabase"
)

const BufferSize = 1024 * 1024

type Service interface {
	GetUserFile(key string, name string) (*model.File, error)
}

type service struct {
	config       *config.Config
	dbMaster     *database.Database
	dbReplica    *database.Database
	mongodb      *mongodatabase.DBConfig
	cache        *cache.Cache
	fileStore    model.FileStorage
	tmpFileStore model.FileStorage
}

// NewtService create new storage service
func NewService(repos *model.Repos, conf *config.Config) Service {
	svc := &service{
		config:       conf,
		dbMaster:     repos.MasterDB,
		dbReplica:    repos.ReplicaDB,
		mongodb:      repos.MongoDB,
		cache:        repos.Cache,
		fileStore:    repos.Storage,
		tmpFileStore: repos.TmpStorage,
	}
	return svc
}

func (s *service) GetUserFile(key string, name string) (*model.File, error) {
	url, err := s.fileStore.GetPresignedDownloadURL(key, name)
	if err != nil {
		return nil, err
	} else if url != "" {
		return &model.File{
			Name:     name,
			Filename: url,
		}, nil
	}
	return s.fileStore.GetFile(key, name)
}
