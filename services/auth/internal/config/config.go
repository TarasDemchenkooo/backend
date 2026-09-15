package config

type Config struct {
	App      App
	HTTP     HTTP
	Admin    Admin
	Startup  Startup
	Shutdown Shutdown
	Log      Log
	Tracing  Tracing
	Metrics  Metrics
	Postgres Postgres
	Redis    Redis
	Auth     Auth
}

func Load() (Config, error) {
	postgres, err := loadPostgres()
	if err != nil {
		return Config{}, err
	}

	redis, err := loadRedis()
	if err != nil {
		return Config{}, err
	}

	tracing, err := loadTracing()
	if err != nil {
		return Config{}, err
	}

	auth, err := loadAuth()
	if err != nil {
		return Config{}, err
	}

	startup, err := loadStartup()
	if err != nil {
		return Config{}, err
	}

	shutdown, err := loadShutdown()
	if err != nil {
		return Config{}, err
	}

	return Config{
		App:      loadApp(),
		HTTP:     loadHTTP(),
		Admin:    loadAdmin(),
		Startup:  startup,
		Shutdown: shutdown,
		Log:      loadLog(),
		Tracing:  tracing,
		Metrics:  loadMetrics(),
		Postgres: postgres,
		Redis:    redis,
		Auth:     auth,
	}, nil
}
