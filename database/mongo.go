package database

import (
	"context"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	Client         *mongo.Client
	RestCollection *mongo.Collection
	UserCollection *mongo.Collection
	CardCollection *mongo.Collection
)

// Initialize connects to the Mongo cluster
func Initialize() {
	log.Info("BB: Initializing database service")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	c, err := mongo.Connect(ctx, options.Client().ApplyURI(os.Getenv("M_URL")))
	if err != nil {
		panic(err)
	}

	Client = c

	RestCollection = Client.Database(os.Getenv("M_DB")).Collection("restaurants")
	UserCollection = Client.Database(os.Getenv("M_DB")).Collection("users")
	CardCollection = Client.Database(os.Getenv("M_DB")).Collection("cards")

	err = Client.Ping(ctx, nil)
	if err != nil {
		log.Error(err)
	}

	log.Info("BB: Connected to the following databases:")
	log.Info(Client.ListDatabaseNames(ctx, bson.D{}))
}
