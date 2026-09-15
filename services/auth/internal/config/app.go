package config

type App struct {
	Name        string
	Version     string
	Environment string
}

func loadApp() App {
	return App{
		Name:        envOr("OTEL_SERVICE_NAME", "auth"),
		Version:     envOr("SERVICE_VERSION", "dev"),
		Environment: envOr("APP_ENV", "local"),
	}
}
