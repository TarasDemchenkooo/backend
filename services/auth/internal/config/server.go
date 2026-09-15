package config

type HTTP struct {
	Addr string
}

func loadHTTP() HTTP {
	return HTTP{Addr: envOr("HTTP_ADDR", ":8080")}
}

type Admin struct {
	Addr string
}

func loadAdmin() Admin {
	return Admin{Addr: envOr("ADMIN_ADDR", ":9090")}
}
