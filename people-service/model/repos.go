package model

import (
	"people-service/cache"
	"people-service/database"
	"people-service/mongodatabase"
	authProtobuf "people-service/proto/v1/pb/authentication"
)

// Repos container to hold handles for cache / db repos
type Repos struct {
	MasterDB          *database.Database
	ReplicaDB         *database.Database
	Cache             *cache.Cache
	MongoDB           *mongodatabase.DBConfig
	Storage           FileStorage
	TmpStorage        FileStorage
	AuthServiceClient authProtobuf.AuthServiceClient
}
