package model

import (
	"people-service/cache"
	"people-service/database"
	"people-service/mongodatabase"
)

// Repos container to hold handles for cache / db repos
type Repos struct {
	MasterDB  *database.Database
	ReplicaDB *database.Database
	Cache     *cache.Cache
	MongoDB   *mongodatabase.DBConfig
}