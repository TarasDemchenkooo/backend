package crypto

import (
	"fmt"

	"github.com/TarasDemchenkooo/backend/services/auth/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

type BcryptHasher struct {
	cost int
}

func NewBcryptHasher(cost int) *BcryptHasher {
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		cost = bcrypt.DefaultCost
	}

	return &BcryptHasher{cost: cost}
}

func (h *BcryptHasher) Hash(password domain.RawPassword) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password.String()), h.cost)
	if err != nil {
		return "", fmt.Errorf("bcrypt generate: %w", err)
	}

	return string(hash), nil
}
