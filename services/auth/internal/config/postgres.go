package config

type Postgres struct {
	DSN string
}

func loadPostgres() (Postgres, error) {
	dsn, err := mustEnv("POSTGRES_DSN")
	if err != nil {
		return Postgres{}, err
	}

	return Postgres{DSN: dsn}, nil
}
