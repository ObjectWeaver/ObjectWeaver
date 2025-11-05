package infrastructure

import (
	"math/rand"
	"time"
)

func GenerateSeed() *int {
	rand.Seed(time.Now().UnixNano())
	seed := rand.Intn(1000000)
	return &seed
}