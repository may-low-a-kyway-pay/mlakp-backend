package auth

import "golang.org/x/crypto/bcrypt"

type BcryptHasher struct {
	Cost int
}

func (h BcryptHasher) HashPassword(password string) (string, error) {
	cost := h.Cost
	if cost == 0 {
		// Zero uses bcrypt's default; non-zero supports tests and explicit tuning.
		cost = bcrypt.DefaultCost
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}

func (h BcryptHasher) ComparePassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
