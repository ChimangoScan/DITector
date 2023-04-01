package db

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
)

type DockerDB struct {
	// 传入数据库
	DSN string
	db  *sql.DB
}

func NewDockerDB(dsn string) (*DockerDB, error) {
	d := DockerDB{DSN: dsn}
	var err error
	d.db, err = sql.Open("mysql", dsn)
	return &d, err
}
