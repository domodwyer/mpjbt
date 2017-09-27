package postgres

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/url"
	"runtime"
	"strconv"

	"github.com/domodwyer/mpjbt/idgen"
	"github.com/domodwyer/mpjbt/record"
	_ "github.com/lib/pq"
)

// FuncProvider implements dbProvider for PostgreSQL.
//
// Methods defined on the FuncProvider perform JSON (un)marshalling where
// appropriate to ensure a fair comparison as the mgo driver does this
// automatically. Ideally we'd want ot just read the data and not perform an
// (un)marshalling on the client side but this is a fair compromise.
type FuncProvider struct {
	DB        *sql.DB
	TableName string
}

// InsertRecord generates a new random record and inserts it with an ID provided
// by id.GetNew as a JSON-encoded string.
func (p *FuncProvider) InsertRecord(data *record.Person, id idgen.Generator, rnd *rand.Rand) bool {
	data.Randomise(rnd)
	data.ID = id.GetNew()

	jsonData, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}

	_, err = p.DB.Exec("INSERT INTO "+p.TableName+" (data) VALUES ($1)", string(jsonData))
	if err != nil {
		log.Println(err)
		return false
	}

	return true
}

// UpdateRecord attempts to update the record with ID returned by
// id.GetExisting.
//
// The balance field is changed to a random value from rnd using jsonb_set.
func (p *FuncProvider) UpdateRecord(_ *record.Person, id idgen.Generator, rnd *rand.Rand) bool {
	recordID := id.GetExisting()
	_, err := p.DB.Exec(
		"UPDATE "+p.TableName+" SET data=jsonb_set(data, '{balance}', $1::jsonb, false) where data->'id'=$2;",
		strconv.FormatFloat(float64(rnd.Float32()), 'f', -1, 32),
		recordID,
	)
	if err != nil {
		log.Println(recordID, err)
		return false
	}
	return true
}

// ReadRecord attempts to fetch the record with an ID returned by
// id.GetExisting.
func (p *FuncProvider) ReadRecord(_ *record.Person, id idgen.Generator, _ *rand.Rand) bool {
	recordID := id.GetExisting()

	var rawData []byte
	err := p.DB.QueryRow("SELECT data FROM "+p.TableName+" WHERE data->'id'=$1", recordID).Scan(&rawData)
	if err != nil {
		log.Println(recordID, err)
		return false
	}

	var data = &record.Person{}
	if err := json.Unmarshal(rawData, &data); err != nil {
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
	rows, err := p.DB.Query("SELECT data FROM " + p.TableName + " WHERE (data->'age') > '45' AND (data->'age') < '75'")
	if err != nil {
		log.Println(err)
		return false
	}
	defer rows.Close()

	var rawData []byte
	var data = &record.Person{}
	for rows.Next() {
		if err := rows.Scan(&rawData); err != nil {
			log.Println(err)
			return false
		}

		if err := json.Unmarshal(rawData, &data); err != nil {
			log.Println(err)
			return false
		}
	}

	return true
}

// ReadMostRecentRecord fetches the most recently inserted record by performing
// a sort on the ID field, and limiting the results to a single record.
func (p *FuncProvider) ReadMostRecentRecord(_ *record.Person, _ idgen.Generator, _ *rand.Rand) bool {
	var rawData []byte
	err := p.DB.QueryRow("SELECT data FROM " + p.TableName + " ORDER BY data->'id' DESC LIMIT 1").Scan(&rawData)
	if err != nil {
		log.Println(err)
		return false
	}

	var data = &record.Person{}
	if err := json.Unmarshal(rawData, &data); err != nil {
		log.Println(err)
		return false
	}

	return true
}

// GetMaxID returns the highest ID in the table.
func (p *FuncProvider) GetMaxID() (uint64, error) {
	var count uint64
	if err := p.DB.QueryRow("SELECT data->'id' FROM " + p.TableName + " ORDER BY data->'id' DESC LIMIT 1").Scan(&count); err != nil || count == 0 {
		return 0, fmt.Errorf("no existing data? error = %v, count = %d", err, count)
	}

	return count, nil
}

// NewProvider returns an instance of FuncProvider.
func NewProvider(endpoint *url.URL, tableName string) (*FuncProvider, error) {
	// Connect to postgres
	db, err := sql.Open("postgres", endpoint.String())
	if err != nil {
		return nil, err
	}

	// Ensure the connection is alive
	if err := db.Ping(); err != nil {
		return nil, err
	}

	// BUG: work around Go garbage collection bug:
	//
	// https://github.com/golang/go/issues/21056
	runtime.GOMAXPROCS(2)

	// DB func provider
	return &FuncProvider{
		DB:        db,
		TableName: tableName,
	}, nil
}
