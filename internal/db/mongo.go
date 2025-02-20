package db

import (
	"context"
	"log"
	"time"

	"github.com/alexbilevskiy/tgWatch/internal/config"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type DbAccountData struct {
	Id       int64
	Phone    string
	DbPrefix string
	DataDir  string
	Status   string
}

func NewClient(cfg *config.Config) *mongo.Client {
	rb := bson.NewRegistryBuilder()

	registry := rb.Build()
	clientOptions := options.Client().ApplyURI(cfg.Mongo["uri"]).SetRegistry(registry)

	var err error
	mctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(mctx, clientOptions)
	if err != nil {
		log.Fatalf("Mongo error: %s", err)
		return nil
	}

	return client
}
