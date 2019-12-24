package crud

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/globalsign/mgo"
)

// MongoClaimsCollection is the name of the claims collection.
const MongoClaimsCollection = "cnab_claims"

type mongoDBStore struct {
	session    *mgo.Session
	collection *mgo.Collection
	dbName     string
}

type doc struct {
	Name string `json:"name"`
	Data []byte `json:"data"`
}

// NewMongoDBStore creates a new storage engine that uses MongoDB
//
// The URL provided must point to a MongoDB server and database.
func NewMongoDBStore(url string) (Store, error) {
	session, err := mgo.Dial(url)
	if err != nil {
		return nil, err
	}

	dbn, err := parseDBName(url)
	if err != nil {
		return nil, err
	}

	return &mongoDBStore{
		session:    session,
		collection: session.DB(dbn).C(MongoClaimsCollection),
		dbName:     dbn,
	}, nil
}

func (s *mongoDBStore) List() ([]string, error) {
	var res []doc
	if err := s.collection.Find(nil).All(&res); err != nil {
		return []string{}, wrapErr(err)
	}
	buf := []string{}
	for _, v := range res {
		buf = append(buf, v.Name)
	}
	return buf, nil
}

func (s *mongoDBStore) Store(name string, data []byte) error {
	return wrapErr(s.collection.Insert(doc{name, data}))
}
func (s *mongoDBStore) Read(name string) ([]byte, error) {
	res := doc{}
	if err := s.collection.Find(map[string]string{"name": name}).One(&res); err != nil {
		if err == mgo.ErrNotFound {
			return nil, ErrRecordDoesNotExist
		}
		return []byte{}, wrapErr(err)
	}
	return res.Data, nil
}
func (s *mongoDBStore) Delete(name string) error {
	return wrapErr(s.collection.Remove(map[string]string{"name": name}))
}

func wrapErr(err error) error {
	if err == nil {
		return err
	}
	return fmt.Errorf("mongo storage error: %s", err)
}

func parseDBName(dialStr string) (string, error) {
	u, err := url.Parse(dialStr)
	if err != nil {
		return "", err
	}
	if u.Path != "" {
		return strings.TrimPrefix(u.Path, "/"), nil
	}
	// If this returns empty, then the driver is supposed to substitute in the
	// default database.
	return "", nil
}
