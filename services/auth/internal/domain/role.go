package domain

type Role string

const (
	RoleAthlete Role = "athlete"
	RoleTrainer Role = "trainer"
)

func NewRole(s string) (Role, error) {
	switch Role(s) {
	case RoleAthlete:
		return RoleAthlete, nil
	case RoleTrainer:
		return RoleTrainer, nil
	default:
		return "", ErrInvalidRole
	}
}

func (r Role) String() string { return string(r) }
