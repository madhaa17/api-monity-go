package config

import (
	"strings"
	"testing"
)

func validConfig(env string) *Config {
	return &Config{
		App:      AppConfig{Env: env},
		Database: DatabaseConfig{User: "u", Name: "db"},
		Jwt: JwtConfig{
			Secret:        strings.Repeat("a", minSecretLen),
			RefreshSecret: strings.Repeat("b", minSecretLen),
		},
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		env     string
		mutate  func(c *Config)
		wantErr string // empty = valid
	}{
		{name: "valid production", env: "production", mutate: func(c *Config) {}},
		{name: "short secrets allowed in development", env: "development", mutate: func(c *Config) {
			c.Jwt.Secret, c.Jwt.RefreshSecret = "dev", "dev-refresh"
		}},
		{name: "missing JWT_SECRET", env: "development", mutate: func(c *Config) { c.Jwt.Secret = "" }, wantErr: "JWT_SECRET must be set"},
		{name: "missing JWT_REFRESH_SECRET", env: "development", mutate: func(c *Config) { c.Jwt.RefreshSecret = "" }, wantErr: "JWT_REFRESH_SECRET must be set"},
		{name: "same secrets", env: "development", mutate: func(c *Config) { c.Jwt.RefreshSecret = c.Jwt.Secret }, wantErr: "must be different"},
		{name: "short secret in production", env: "production", mutate: func(c *Config) { c.Jwt.Secret = "short" }, wantErr: "at least 32 characters"},
		{name: "missing database", env: "development", mutate: func(c *Config) { c.Database.Name = "" }, wantErr: "DATABASE_USER and DATABASE_NAME"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := validConfig(tt.env)
			tt.mutate(c)
			err := c.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() = %v, want nil", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Validate() = %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestLoadHasNoJWTFallback(t *testing.T) {
	t.Setenv("JWT_SECRET", "")
	t.Setenv("JWT_REFRESH_SECRET", "")
	t.Chdir(t.TempDir()) // no .env to load
	c, _ := Load()
	if c.Jwt.Secret != "" || c.Jwt.RefreshSecret != "" {
		t.Fatalf("Load() used a fallback JWT secret: %q / %q", c.Jwt.Secret, c.Jwt.RefreshSecret)
	}
	if c.PriceAPI.CryptoAPI != CoinGeckoAPI || c.PriceAPI.StockAPI != YahooAPI || c.PriceAPI.FXAPI != ExchangeAPI {
		t.Fatalf("price API URLs not set from constants: %+v", c.PriceAPI)
	}
}
