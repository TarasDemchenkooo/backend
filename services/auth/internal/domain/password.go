package domain

type RawPassword string

func NewRawPassword(pwd string) (RawPassword, error) {
	if len([]rune(pwd)) < 8 {
		return "", ErrWeakPassword
	}

	return RawPassword(pwd), nil
}

func (p RawPassword) String() string { return string(p) }
