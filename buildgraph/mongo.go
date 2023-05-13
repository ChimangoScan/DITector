package buildgraph

import "go.mongodb.org/mongo-driver/mongo"

// mongo.go 用于操作mongodb

var mongoClient *mongo.Client
var mongoRepositoryCollection *mongo.Collection

func InsertRepositoryToMongo(repo *Repository) {

}
