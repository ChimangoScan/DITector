package db

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"log"
)

func init() {
	dsn := "docker:docker@%s/%s"

	// 初始化创建新的database，命名为dockerhub
	// 默认data source name
	db, err := sql.Open("mysql", fmt.Sprintf(dsn, "tcp(localhost:3306)", ""))
	if err != nil {
		log.Fatalln("[ERROR] Open mysql failed with err: ", err)
	}
	defer db.Close()
	fmt.Println("Open mysql Success.")
	if err := db.Ping(); err != nil {
		log.Fatalln("[ERROR] Ping mysql database failed with err: ", err)
	}
	fmt.Println("Ping mysql Success.")

	createDatabase := `CREATE DATABASE IF NOT EXISTS dockerhub`
	_, err = db.Exec(createDatabase)
	if err != nil {
		log.Fatalln("[ERROR] CREATE DATABASE dockerhub failed with err: ", err)
	} else {
		fmt.Println("Create db dockerhub success.")
	}

	// 初始化dockerhub数据库内的数据表
	db2, err := sql.Open("mysql", fmt.Sprintf(dsn, "tcp(localhost:3306)", "dockerhub"))
	if err != nil {
		log.Fatalln("[ERROR] Open DATABASE dockerhub failed with err: ", err)
	}
	defer db2.Close()
	fmt.Println("Open DATABASE dockerhub Success.")
	if err := db2.Ping(); err != nil {
		log.Fatalln("[ERROR] Ping database dockerhub failed with err: ", err)
	}
	fmt.Println("Ping database dockerhub Success.")

	// 创建keywords表
	createKeywords := `
CREATE TABLE IF NOT EXISTS keywords
(
    name VARCHAR(255) UNIQUE 
);`
	_, err = db2.Exec(createKeywords)
	if err != nil {
		log.Fatalln("[ERROR] CREATE TABLE keywords failed with err: ", err)
	} else {
		fmt.Println("CREATE TABLE keywords success.")
	}

	// 创建repository表
	createRepository := `
CREATE TABLE IF NOT EXISTS repository
(
    user VARCHAR(255),
    name VARCHAR(255),
    namespace VARCHAR(255),
    repository_type VARCHAR(255),
    description TEXT,
	flag TINYINT,
	star_count MEDIUMINT,
	pull_count BIGINT,
	last_updated TIMESTAMP,
	date_registered TIMESTAMP,
	full_description LONGTEXT,
	media_types TEXT,
	content_types TINYTEXT,
	PRIMARY KEY (user,name)
);`
	_, err = db2.Exec(createRepository)
	if err != nil {
		log.Fatalln("[ERROR] CREATE TABLE repository failed with err: ", err)
	} else {
		fmt.Println("CREATE TABLE repository success.")
	}

}
