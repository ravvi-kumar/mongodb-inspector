package mongo

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Connector struct {
	client *mongo.Client
}

func NewConnector(ctx context.Context, connectionStr string) (*Connector, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(connectionStr))
	if err != nil {
		return nil, fmt.Errorf("mongo connect: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("mongo ping: %w", err)
	}

	return &Connector{client: client}, nil
}

func (c *Connector) Close(ctx context.Context) error {
	return c.client.Disconnect(ctx)
}

func (c *Connector) ListDatabases(ctx context.Context) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	result, err := c.client.ListDatabases(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("list databases: %w", err)
	}

	names := make([]string, 0, len(result.Databases))
	for _, db := range result.Databases {
		if db.Name == "admin" || db.Name == "local" || db.Name == "config" {
			continue
		}
		names = append(names, db.Name)
	}
	return names, nil
}

func (c *Connector) ListCollections(ctx context.Context, dbName string) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cursor, err := c.client.Database(dbName).ListCollections(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("list collections: %w", err)
	}
	defer cursor.Close(ctx)

	var names []string
	for cursor.Next(ctx) {
		var coll struct {
			Name string `bson:"name"`
			Type string `bson:"type"`
		}
		if err := cursor.Decode(&coll); err != nil {
			return nil, fmt.Errorf("decode collection: %w", err)
		}
		if coll.Type == "collection" {
			names = append(names, coll.Name)
		}
	}
	return names, cursor.Err()
}

func (c *Connector) Client() *mongo.Client {
	return c.client
}

func (c *Connector) Database(dbName string) *mongo.Database {
	return c.client.Database(dbName)
}
