package db

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"sync"
)

type DockerDB struct {
	Path string
	db   sql.DB
	rwl  sync.RWMutex
}
