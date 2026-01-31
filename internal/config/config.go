package config

import "strconv"

type Config struct {
	App      AppConfig
	Database DatabaseConfig
	Jwt      JwtConfig
}

type AppConfig struct {
	Env  string
	Port string
}

type DatabaseConfig struct {
	Host               string
	Port               string
	User               string
	Password           string
	Name               string
	MaxOpenConnections int
	MaxIdleConnections int
}

type JwtConfig struct {
	Secret         string
	ExpirationTime string
}

func Load() (*Config, error) {
	_ = loadEnv()

	maxOpen, _ := strconv.Atoi(getEnv("DATABASE_MAX_OPEN_CONNECTIONS", "10"))
	maxIdle, _ := strconv.Atoi(getEnv("DATABASE_MAX_IDLE_CONNECTIONS", "10"))

	return &Config{
		App: AppConfig{
			Env:  getEnv("APP_ENV", "development"),
			Port: getEnv("APP_PORT", "8080"),
		},
		Database: DatabaseConfig{
			Host:               getEnv("DATABASE_HOST", "localhost"),
			Port:               getEnv("DATABASE_PORT", "5432"),
			User:               getEnv("DATABASE_USER", ""),
			Password:           getEnv("DATABASE_PASSWORD", ""),
			Name:               getEnv("DATABASE_NAME", ""),
			MaxOpenConnections: maxOpen,
			MaxIdleConnections: maxIdle,
		},
		Jwt: JwtConfig{
			Secret:         getEnv("JWT_SECRET", "secret"),
			ExpirationTime: getEnv("JWT_EXPIRATION_TIME", "1h"),
		},
	}, nil
}

func (c *DatabaseConfig) DSN() string {
	return "postgres://" + c.User + ":" + c.Password + "@" + c.Host + ":" + c.Port + "/" + c.Name + "?sslmode=disable"
}
