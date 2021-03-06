package tests

import (
	"time"

	"github.com/google/uuid"

	"github.com/neuronlabs/neuron-extensions/neurogns/internal/tests/external"
)

//go:generate neurogns models methods --format=goimports --single-file .
//go:generate neurogns collections --format=goimports --single-file .

// User is testing model.
type User struct {
	ID          uuid.UUID
	CreatedAt   time.Time
	DeletedAt   *time.Time
	Name        *string
	Age         int
	IntArray    []int
	Bytes       []byte
	PtrBytes    *[]byte
	Wrapped     external.Int
	PtrWrapped  *external.Int
	External    *external.Model
	FavoriteCar Car
	Cars        []*Car
	Sons        []User
	Sister      *User
}

// Car is the test model for generator.
type Car struct {
	ID     string
	Plates string
}
