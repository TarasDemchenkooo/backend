package config

import "time"

type Auth struct {
	VerificationCodeTTL time.Duration
	BcryptCost          int
}

func loadAuth() (Auth, error) {
	ttl, err := envDuration("VERIFICATION_CODE_TTL", 15*time.Minute)
	if err != nil {
		return Auth{}, err
	}

	bcryptCost, err := envInt("BCRYPT_COST", 10)
	if err != nil {
		return Auth{}, err
	}

	return Auth{
		VerificationCodeTTL: ttl,
		BcryptCost:          bcryptCost,
	}, nil
}
