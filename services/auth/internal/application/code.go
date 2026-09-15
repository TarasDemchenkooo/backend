package application

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

const verificationCodeDigits = 6

func newVerificationCode() (string, error) {
	upperBound := big.NewInt(1)
	for range verificationCodeDigits {
		upperBound.Mul(upperBound, big.NewInt(10))
	}

	n, err := rand.Int(rand.Reader, upperBound)
	if err != nil {
		return "", fmt.Errorf("generate verification code: %w", err)
	}

	return fmt.Sprintf("%0*d", verificationCodeDigits, n), nil
}
