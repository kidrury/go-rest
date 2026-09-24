package config

import (
	"errors"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr        string
	Environment     string
	JWTSecret       string
	JWTIssuer       string
	JWTAudience     string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	RefreshIdleTTL  time.Duration
	DatabaseURL     string
	TLSEnabled      bool
	TLSCertFile     string
	TLSKeyFile      string
	ShutdownTimeout time.Duration
}

func envOrDefault(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func requiredEnv(key string) (string, error) {
	val := os.Getenv(key)
	if val == "" {
		return "", errors.New(key + " is required")
	}
	return val, nil
}

func environmentEnv() (string, error) {
	val := os.Getenv("ENV")

	switch val {
	case "":
		return "development", nil
	case "development", "production":
		return val, nil
	default:
		return "", errors.New("ENVIRONMENT must be development or production")
	}
}

func Load() (*Config, error) {
	jwtSecret, err := requiredEnv("JWT_SECRET")
	if err != nil {
		return &Config{}, err
	}

	jwtIssuer := envOrDefault("JWT_ISSUER", "issued_by_alireza")

	jwtAudience := envOrDefault("JWT_AUDIENCE", "shmucks_who_use")

	accessTokenTTL, err := time.ParseDuration(envOrDefault("ACCESS_TOKEN_TTL_M", "1m"))
	if err != nil {
		return &Config{}, errors.New("ACCESS_TOKEN_TTL_M must be a valid duration")
	}

	refreshTokenTTL, err := time.ParseDuration(envOrDefault("REFRESH_TOKEN_TTL", "720h"))
	if err != nil {
		return &Config{}, errors.New("REFRESH_TOKEN_TTL must be a valid duration")
	}

	refreshIdleTTL, err := time.ParseDuration(envOrDefault("REFRESH_IDLE_TTL", "168h"))
	if err != nil {
		return &Config{}, errors.New("REFRESH_IDLE_TTL must be a valid duration")
	}

	databaseURL, err := requiredEnv("DATABASE_URL")
	if err != nil {
		return &Config{}, err
	}

	tlsEnabled, err := strconv.ParseBool(envOrDefault("TLS_ENABLED", "false"))
	if err != nil {
		return &Config{}, errors.New("TLS_ENABLED must be true or false")
	}

	shutdownTimeout, err := time.ParseDuration(envOrDefault("SHUTDOWN_TIMEOUT_S", "10s"))
	if err != nil {
		return &Config{}, errors.New("SHUTDOWN_TIMEOUT_S must be a valid duration")
	}

	environment, err := environmentEnv()
	if err != nil {
		return &Config{}, errors.New("SHUTDOWN_TIMEOUT_S must be a valid duration")
	}

	config := &Config{
		HTTPAddr:        envOrDefault("HTTP_ADDR", ":8091"),
		Environment:     environment,
		JWTSecret:       jwtSecret,
		JWTIssuer:       jwtIssuer,
		JWTAudience:     jwtAudience,
		AccessTokenTTL:  accessTokenTTL,
		RefreshTokenTTL: refreshTokenTTL,
		RefreshIdleTTL:  refreshIdleTTL,
		DatabaseURL:     databaseURL,
		TLSEnabled:      tlsEnabled,
		TLSCertFile:     envOrDefault("TLS_CERT_FILE", "server.crt"),
		TLSKeyFile:      envOrDefault("TLS_KEY_FILE", "server.key"),
		ShutdownTimeout: shutdownTimeout,
	}

	return config, nil
}
