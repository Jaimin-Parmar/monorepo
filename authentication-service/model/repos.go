package model

import (
	"authentication-service/cache"
	"authentication-service/database"
	"authentication-service/mongodatabase"
	acProtobuf "authentication-service/proto/v1/pb/account"
)

// Repos container to hold handles for cache / db repos
type Repos struct {
	MasterDB             *database.Database
	ReplicaDB            *database.Database
	Cache                *cache.Cache
	MongoDB              *mongodatabase.DBConfig
	AccountServiceClient acProtobuf.AccountServiceClient
}
