package record

import (
	"math/rand"
	"time"
)

// Person defines a semi-realistic data structure to be stored in the database.
//
// Person was chosen to have a wide range of field types.
type Person struct {
	ID          uint64    `bson:"_id"           json:"id"`
	Name        string    `bson:"name"          json:"name"`
	Address     []Address `bson:"addresses"     json:"addresses"`
	PhoneNumber string    `bson:"phone_number"  json:"phone_number"`
	DateOfBirth time.Time `bson:"dob"           json:"dob"`
	Age         uint32    `bson:"age"           json:"age"`
	Balance     float64   `bson:"balance"       json:"balance"`
	Enabled     bool      `bson:"enabled"       json:"enabled"`
	Counter     int32     `bson:"counter"       json:"counter"`
	Padding     []byte    `bson:"padding"       json:"padding"`
}

// Address defines a sub-document within Person.
type Address struct {
	Number uint8
	Line1  string
	Line2  string
}

// TODO: change randomisation to use masking and read one block from the
// rand.Rand per Randomise call.

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func (p *Person) randStringBytesRmndr(rnd *rand.Rand, n int) string {
	x := rnd.Int63()
	b := make([]byte, int(x)%n)
	for i := range b {
		b[i] = letterBytes[x%int64(len(letterBytes))]
		x = rnd.Int63()
	}
	return string(b)
}

// Randomise uses rnd to populate p - existing data is overwrote.
func (p *Person) Randomise(rnd *rand.Rand) {
	rnd.Read(p.Padding)
	p.Name = p.randStringBytesRmndr(rnd, 50)

	n := rnd.Uint64()
	p.Address = make([]Address, rnd.Intn(5))
	for i := range p.Address {
		p.Address[i].Number = uint8(n)
		p.Address[i].Line1 = p.randStringBytesRmndr(rnd, 30)
		p.Address[i].Line2 = p.randStringBytesRmndr(rnd, 30)
	}

	p.PhoneNumber = p.randStringBytesRmndr(rnd, 30)
	p.DateOfBirth = time.Now()
	p.Age = uint32(n)
	p.Balance = rnd.NormFloat64()
	p.Counter = int32(n)
}
