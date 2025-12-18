package db

import (
	"context"
	"fmt"
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

func NewClient(ctx context.Context, cfg *config.Config) (*mongo.Client, error) {
	rb := bson.NewRegistryBuilder()

	registry := rb.Build()
	clientOptions := options.Client().ApplyURI(cfg.Mongo["uri"]).SetRegistry(registry)

	var err error
	mctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(mctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("mongo connect: %w", err)
	}

	return client, nil
}
