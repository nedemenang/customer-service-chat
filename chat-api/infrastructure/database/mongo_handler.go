package database

import (
	"chat-api/adapter/repository"
	"context"

	"log"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type mongoHandler struct {
	db     *mongo.Database
	client *mongo.Client
}

func NewMongoHandler(c *config) (*mongoHandler, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.ctxTimeout)
	defer cancel()

	uri := c.uri

	clientOpts := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		log.Fatal(err)
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}

	return &mongoHandler{
		db:     client.Database(c.database),
		client: client,
	}, nil
}

func (mgo mongoHandler) EnsureIndex(
	ctx context.Context,
	collection string,
	keys interface{},
	unique bool,
) error {
	indexOptions := options.Index()
	indexOptions.SetUnique(unique)
	_, err := mgo.db.Collection(collection).Indexes().CreateOne(ctx, mongo.IndexModel{
		Options: indexOptions,
		Keys:    keys,
	})
	return err
}

func (mgo mongoHandler) Store(ctx context.Context, collection string, data interface{}) error {
	if _, err := mgo.db.Collection(collection).InsertOne(ctx, data); err != nil {
		return err
	}

	return nil
}

func (mgo mongoHandler) Update(ctx context.Context, collection string, query interface{}, update interface{}) error {
	if _, err := mgo.db.Collection(collection).UpdateOne(ctx, query, update); err != nil {
		return err
	}

	return nil
}

func (mgo mongoHandler) FindCount(ctx context.Context, collection string, query interface{}) (int64, error) {
	count, err := mgo.db.Collection(collection).CountDocuments(ctx, query)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (mgo mongoHandler) FindAll(ctx context.Context, collection string, query interface{}, result interface{}, findOptions *options.FindOptions) error {
	cur, err := mgo.db.Collection(collection).Find(ctx, query, findOptions)
	if err != nil {
		return err
	}

	defer cur.Close(ctx)
	if err = cur.All(ctx, result); err != nil {
		return err
	}

	if err := cur.Err(); err != nil {
		return err
	}

	return nil
}

func (mgo mongoHandler) Aggregate(ctx context.Context, collection string, query interface{}, result interface{}, findOptions *options.AggregateOptions) error {
	cur, err := mgo.db.Collection(collection).Aggregate(ctx, query, findOptions)
	if err != nil {
		return err
	}

	defer cur.Close(ctx)
	if err = cur.All(ctx, result); err != nil {
		return err
	}

	if err := cur.Err(); err != nil {
		return err
	}

	return nil
}

func (mgo mongoHandler) FindOne(
	ctx context.Context,
	collection string,
	query interface{},
	projection interface{},
	result interface{},
) error {
	var err = mgo.db.Collection(collection).
		FindOne(
			ctx,
			query,
			options.FindOne().SetProjection(projection),
		).Decode(result)
	if err != nil {
		return err
	}

	return nil
}

func (mgo *mongoHandler) StartSession() (repository.Session, error) {
	session, err := mgo.client.StartSession()
	if err != nil {
		log.Fatal(err)
	}

	return newMongoHandlerSession(session), nil
}

type mongoDBSession struct {
	session mongo.Session
}

func newMongoHandlerSession(session mongo.Session) *mongoDBSession {
	return &mongoDBSession{session: session}
}

func (m *mongoDBSession) WithTransaction(ctx context.Context, fn func(context.Context) error) error {
	callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
		err := fn(sessCtx)
		if err != nil {
			return nil, err
		}
		return nil, nil
	}

	_, err := m.session.WithTransaction(ctx, callback)
	if err != nil {
		return err
	}

	return nil
}

func (m *mongoDBSession) EndSession(ctx context.Context) {
	m.session.EndSession(ctx)
}