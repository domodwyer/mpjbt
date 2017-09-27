package mongo

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/url"
	"strconv"
	"strings"

	"github.com/domodwyer/mpjbt/idgen"
	"github.com/domodwyer/mpjbt/record"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

// FuncProvider implements dbProvider for MongoDB.
//
// Methods are safe for concurrent use, however the struct fields are not.
type FuncProvider struct {
	Session    *mgo.Session
	Collection string
}

// InsertRecord generates a new random record and inserts it with an ID provided
// by id.GetNew
func (p *FuncProvider) InsertRecord(data *record.Person, id idgen.Generator, rnd *rand.Rand) bool {
	conn := p.Session.Copy()
	defer conn.Close()

	data.Randomise(rnd)
	data.ID = id.GetNew()

	if err := conn.DB("").C(p.Collection).Insert(data); err != nil {
		log.Println(err)
		return false
	}

	return true
}

// UpdateRecord attempts to update the record with ID returned by
// id.GetExisting.
//
// The balance field is changed to a random value from rnd.
func (p *FuncProvider) UpdateRecord(_ *record.Person, id idgen.Generator, rnd *rand.Rand) bool {
	conn := p.Session.Copy()
	defer conn.Close()

	recordID := id.GetExisting()
	err := conn.DB("").C(p.Collection).Update(
		bson.M{"_id": recordID},
		bson.M{"$set": bson.M{"balance": rnd.Float32()}},
	)
	if err != nil {
		log.Println(recordID, err)
		return false
	}

	return true
}

// ReadRecord attempts to fetch the record with an ID returned by
// id.GetExisting.
func (p *FuncProvider) ReadRecord(_ *record.Person, id idgen.Generator, rnd *rand.Rand) bool {
	conn := p.Session.Copy()
	defer conn.Close()

	recordID := id.GetExisting()
	query := conn.DB("").
		C(p.Collection).
		Find(bson.M{"_id": recordID}).
		Limit(1)

	var data = &record.Person{}
	if err := query.One(data); err != nil {
		log.Println(recordID, err)
		return false
	}

	return true
}

// ReadRange performs a range query on the age field.
//
// THe query attempts to fetch all records where age is greater than 45 and less
// than 75.
func (p *FuncProvider) ReadRange(_ *record.Person, _ idgen.Generator, _ *rand.Rand) bool {
	conn := p.Session.Copy()
	defer conn.Close()

	filter := bson.M{
		"age": bson.M{
			"$gt": 45,
			"$lt": 75,
		},
	}

	iter := conn.DB("").
		C(p.Collection).
		Find(filter).
		Iter()

	var data = &record.Person{}
	for iter.Next(data) {
		// It does nothing!
		//
		// Make sure we actually read all the data, otherwise it's just the cost
		// of getting a cursor and the inital batch.
	}

	if err := iter.Close(); err != nil {
		log.Println(err)
		return false
	}

	return true
}

// ReadMostRecentRecord fetches the most recently inserted record by performing
// a sort on the ID field, and limiting the results to a single record.
func (p *FuncProvider) ReadMostRecentRecord(_ *record.Person, _ idgen.Generator, _ *rand.Rand) bool {
	conn := p.Session.Copy()
	defer conn.Close()

	query := conn.DB("").
		C(p.Collection).
		Find(bson.M{}).
		Sort("-_id").
		Limit(1)

	var data = &record.Person{}
	if err := query.One(data); err != nil {
		log.Println(err)
		return false
	}

	return true
}

// GetMaxID returns the largest ID in the collection.
func (p *FuncProvider) GetMaxID() (uint64, error) {
	conn := p.Session.Copy()
	defer conn.Close()

	query := conn.DB("").
		C(p.Collection).
		Find(bson.M{}).
		Sort("-_id").
		Limit(1)

	var id = struct {
		ID uint64 `bson:"_id"`
	}{}

	if err := query.One(&id); err != nil {
		return 0, fmt.Errorf("no existing data? %v", err)
	}

	return id.ID, nil
}

// NewProvider returns an instance of FuncProvider.
//
// The following options are available in addition to the usual dial string
// options provided by the gmo driver:
//
// 		readConcern: majority/local/linearizable
// 		writeConcern: majority/<number>
// 		journal: true/false
// 		fsync: true/false
//
func NewProvider(endpoint *url.URL, tableName string) (*FuncProvider, error) {
	// Parse URL options and set flags
	q := endpoint.Query()
	safe := &mgo.Safe{}

	// Read concern
	safe.RMode = strings.ToLower(q.Get("readConcern"))
	switch safe.RMode {
	case "", "majority", "local", "linearizable":
		break
	default:
		return nil, errors.New("unknown readConcern value")
	}
	q.Del("readConcern")

	// Write concern
	w := strings.ToLower(q.Get("writeConcern"))
	switch {
	case w == "" || w == "majority":
		safe.WMode = w
	default:
		w, err := strconv.Atoi(w)
		if err != nil {
			return nil, err
		}
		safe.W = w
	}
	q.Del("writeConcern")

	// Journaling
	switch strings.ToLower(q.Get("journal")) {
	case "", "false", "0":
		safe.J = false
	case "true", "1":
		safe.J = true
	default:
		return nil, errors.New("unknown journal value")
	}
	q.Del("journal")

	// FSync
	switch strings.ToLower(q.Get("fsync")) {
	case "", "false", "0":
		safe.FSync = false
	case "true", "1":
		safe.FSync = true
	default:
		return nil, errors.New("unknown fsync value")
	}
	q.Del("fsync")

	// Reset cleaned dial string to mgo compatible dial string
	endpoint.RawQuery = q.Encode()

	// Connect to mongo
	session, err := mgo.Dial(endpoint.String())
	if err != nil {
		return nil, err
	}

	// Liveness check
	if err := session.Ping(); err != nil {
		return nil, err
	}

	session.SetSafe(safe)
	return &FuncProvider{
		Session:    session,
		Collection: tableName,
	}, nil
}
