package nodes_db

import (
	"context"
	"time"

	"github.com/MurmurationsNetwork/MurmurationsServices/utils/logger"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var (
	Collection *mongo.Collection
	client     *mongo.Client
)

func init() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var err error
	client, err = mongo.Connect(ctx, options.Client().ApplyURI("mongodb://indexer-mongo-srv:27017"))
	if err != nil {
		logger.Panic("error when trying to connect to MongoDB", err)
	}

	ping(client)

	Collection = client.Database("murmurations").Collection("nodes")
}

func Disconnect() {
	logger.Info("trying to disconnect from MongoDB")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Disconnect(ctx); err != nil {
		logger.Panic("error when trying to disconnect from MongoDB", err)
	}
}

func ping(client *mongo.Client) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := client.Ping(ctx, readpref.Primary())
	if err != nil {
		logger.Panic("error when trying to ping the MongoDB", err)
	}
}