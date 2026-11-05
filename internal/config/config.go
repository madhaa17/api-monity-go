package config

import (
	"net/url"
	"strconv"
)

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
	CryptoAPI    string // CoinGecko base URL, e.g. https://api.coingecko.com/api/v3
	CryptoAPIKey string // optional CoinGecko Demo (or Pro) API key
	StockAPI     string
	FXAPI        string // fallback exchange-rate API (open.er-api.com format)
	CacheTTL     int    // in seconds
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
	SSLMode            string // disable | prefer | require | verify-full
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
			SSLMode:            getEnv("DATABASE_SSLMODE", "prefer"),
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
			CryptoAPI:    getEnv("CRYPTO_PRICE_API", "https://api.coingecko.com/api/v3"),
			CryptoAPIKey: getEnv("CRYPTO_PRICE_API_KEY", ""),
			StockAPI:     getEnv("STOCK_PRICE_API", "https://query1.finance.yahoo.com"),
			FXAPI:        getEnv("FX_PRICE_API", "https://open.er-api.com/v6"),
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

// DSN builds a postgres:// URL. User and password are URL-escaped so special
// characters (@, /, :, #) in credentials don't break the connection string.
func (c *DatabaseConfig) DSN() string {
	sslMode := c.SSLMode
	if sslMode == "" {
		sslMode = "prefer"
	}
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(c.User, c.Password),
		Host:     c.Host + ":" + c.Port,
		Path:     "/" + c.Name,
		RawQuery: url.Values{"sslmode": {sslMode}}.Encode(),
	}
	return u.String()
}
