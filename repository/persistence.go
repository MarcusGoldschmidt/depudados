package repository

import (
	"context"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"time"
)

type Persistence struct {
	dbName string
	client *mongo.Client
	db     *mongo.Database
}

func NewPersistence(connectionUrl string, dbName string) (*Persistence, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(connectionUrl))
	if err != nil {
		return nil, err
	}

	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		return nil, err
	}

	return &Persistence{
		client: client,
		db:     client.Database(dbName),
		dbName: dbName,
	}, nil
}

func (p *Persistence) Close() error {
	return p.client.Disconnect(context.Background())
}

func (p *Persistence) PLP() *mongo.Collection {
	return p.db.Collection("plp")
}
