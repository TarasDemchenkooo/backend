package config

type Redis struct {
	Addr     string
	Password string
	DB       int
}

func loadRedis() (Redis, error) {
	db, err := envInt("REDIS_DB", 0)
	if err != nil {
		return Redis{}, err
	}

	return Redis{
		Addr:     envOr("REDIS_ADDR", "localhost:6379"),
		Password: envOr("REDIS_PASSWORD", ""),
		DB:       db,
	}, nil
}
