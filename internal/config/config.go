package config

import "strconv"

type Config struct {
	App       AppConfig
	Database  DatabaseConfig
	Redis     RedisConfig
	Jwt       JwtConfig
	PriceAPI  PriceAPIConfig
	RateLimit RateLimitConfig
	Security  SecurityConfig
}

type RedisConfig struct {
	Host     string
	Port     string
	Username string // Redis 6+ ACL; leave empty for default user
	Password string
	DB       int
}

func (c *RedisConfig) Addr() string {
	if c.Port == "" {
		c.Port = "6379"
	}
	return c.Host + ":" + c.Port
}

func (c *RedisConfig) Enabled() bool {
	return c.Host != ""
}

type RateLimitConfig struct {
	TTLSeconds int // time window in seconds
	Limit      int // max requests per window per client
}

type SecurityConfig struct {
	CORSAllowedOrigins string // comma-separated, e.g. "https://app.example.com,https://admin.example.com"
}

type PriceAPIConfig struct {
	CryptoAPI    string
	CryptoAPIKey string
	StockAPI     string
	CacheTTL     int // in seconds
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
	Secret            string
	ExpirationTime    string
	RefreshSecret     string
	RefreshExpiration string
}

func Load() (*Config, error) {
	_ = loadEnv()

	maxOpen, _ := strconv.Atoi(getEnv("DATABASE_MAX_OPEN_CONNECTIONS", "10"))
	maxIdle, _ := strconv.Atoi(getEnv("DATABASE_MAX_IDLE_CONNECTIONS", "10"))
	cacheTTL, _ := strconv.Atoi(getEnv("REDIS_TTL_PRICE", "60"))
	rateLimitTTL, _ := strconv.Atoi(getEnv("RATE_LIMIT_TTL", "60"))
	rateLimitLimit, _ := strconv.Atoi(getEnv("RATE_LIMIT_LIMIT", "100"))
	redisDB, _ := strconv.Atoi(getEnv("REDIS_DB", "0"))

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
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", ""),
			Port:     getEnv("REDIS_PORT", "6379"),
			Username: getEnv("REDIS_USERNAME", ""),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       redisDB,
		},
		Jwt: JwtConfig{
			Secret:            getEnv("JWT_SECRET", "secret"),
			ExpirationTime:    getEnv("JWT_EXPIRATION_TIME", "1h"),
			RefreshSecret:     getEnv("JWT_REFRESH_SECRET", "refresh_secret"),
			RefreshExpiration: getEnv("JWT_REFRESH_EXPIRATION_TIME", "168h"), // 7d default
		},
		PriceAPI: PriceAPIConfig{
			CryptoAPI:    getEnv("CRYPTO_PRICE_API", "https://pro-api.coinmarketcap.com"),
			CryptoAPIKey: getEnv("CRYPTO_PRICE_API_KEY", ""),
			StockAPI:     getEnv("STOCK_PRICE_API", "https://query1.finance.yahoo.com"),
			CacheTTL:     cacheTTL,
		},
		RateLimit: RateLimitConfig{
			TTLSeconds: rateLimitTTL,
			Limit:      rateLimitLimit,
		},
		Security: SecurityConfig{
			CORSAllowedOrigins: getEnv("CORS_ALLOWED_ORIGINS", "*"),
		},
	}, nil
}

func (c *DatabaseConfig) DSN() string {
	return "postgres://" + c.User + ":" + c.Password + "@" + c.Host + ":" + c.Port + "/" + c.Name + "?sslmode=disable"
}
