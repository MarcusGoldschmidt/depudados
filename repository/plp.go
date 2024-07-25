package repository

import (
	"context"
	"depudados/cameraleg"
	"errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (p *Persistence) GetPlp(ctx context.Context, plpId string) (*cameraleg.PLP, error) {
	result := &cameraleg.PLP{}

	err := p.PLP().FindOne(ctx, bson.M{"id": plpId}).Decode(&result)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}

	return result, err
}

func (p *Persistence) InsertPlp(ctx context.Context, plp *cameraleg.PLP) error {
	_, err := p.PLP().InsertOne(ctx, plp)
	return err
}

func (p *Persistence) UpdatePlp(ctx context.Context, plp *cameraleg.PLP) error {
	update := bson.M{
		"$set": plp,
	}

	_, err := p.PLP().UpdateOne(ctx, bson.M{"id": plp.Id}, update, options.Update().SetUpsert(true))

	return err
}

func (p *Persistence) GetAllToReport(ctx context.Context) ([]*cameraleg.PLP, error) {
	result := make([]*cameraleg.PLP, 0)

	query := bson.M{
		"processadoEm": bson.M{
			"$ne": nil,
		},
		"$and": bson.A{
			bson.M{
				"files": bson.M{
					"$ne": bson.A{},
				},
			},
			bson.M{
				"files": bson.M{
					"$ne": nil,
				},
			},
		},
	}

	cur, err := p.PLP().Find(ctx, query)
	if err != nil {
		return nil, nil
	}

	err = cur.All(ctx, &result)
	if err != nil {
		return nil, err
	}

	return result, err
}
