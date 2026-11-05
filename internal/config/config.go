package config

import (
	"errors"
	"fmt"
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

// Base URLs of the external price providers. They are fixed in code, not read from env;
// PriceAPIConfig keeps them as fields only so tests can point the service at a fake server.
const (
	CoinGeckoAPI = "https://api.coingecko.com/api/v3"
	YahooAPI     = "https://query1.finance.yahoo.com"
	ExchangeAPI  = "https://open.er-api.com/v6" // fallback FX rates when Yahoo FX is unavailable
)

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
			Secret:            getEnv("JWT_SECRET", ""), // required, no fallback: see Validate
			ExpirationTime:    getEnv("JWT_EXPIRATION_TIME", "1h"),
			RefreshSecret:     getEnv("JWT_REFRESH_SECRET", ""),              // required, no fallback: see Validate
			RefreshExpiration: getEnv("JWT_REFRESH_EXPIRATION_TIME", "168h"), // 7d default
		},
		PriceAPI: PriceAPIConfig{
			CryptoAPI:    CoinGeckoAPI,
			CryptoAPIKey: getEnv("CRYPTO_PRICE_API_KEY", ""), // optional; raises the CoinGecko rate limit
			StockAPI:     YahooAPI,
			FXAPI:        ExchangeAPI,
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

// minSecretLen is the shortest JWT secret accepted in production (32 bytes = 256 bits for HS256).
const minSecretLen = 32

// Validate reports every required setting that is missing or unsafe, so the server refuses to
// start instead of running with a guessable JWT secret.
func (c *Config) Validate() error {
	var errs []error
	if c.Database.User == "" || c.Database.Name == "" {
		errs = append(errs, errors.New("DATABASE_USER and DATABASE_NAME must be set"))
	}
	if c.Jwt.Secret == "" {
		errs = append(errs, errors.New("JWT_SECRET must be set"))
	}
	if c.Jwt.RefreshSecret == "" {
		errs = append(errs, errors.New("JWT_REFRESH_SECRET must be set"))
	}
	if c.Jwt.Secret != "" && c.Jwt.Secret == c.Jwt.RefreshSecret {
		// With one key, a refresh token would also pass as an access token.
		errs = append(errs, errors.New("JWT_SECRET and JWT_REFRESH_SECRET must be different"))
	}
	if c.App.Env == "production" {
		if c.Jwt.Secret != "" && len(c.Jwt.Secret) < minSecretLen {
			errs = append(errs, fmt.Errorf("JWT_SECRET must be at least %d characters in production", minSecretLen))
		}
		if c.Jwt.RefreshSecret != "" && len(c.Jwt.RefreshSecret) < minSecretLen {
			errs = append(errs, fmt.Errorf("JWT_REFRESH_SECRET must be at least %d characters in production", minSecretLen))
		}
	}
	return errors.Join(errs...)
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
