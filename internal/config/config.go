package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	EnvLocal      = "local"
	EnvTest       = "test"
	EnvProduction = "production"
)

type Config struct {
	AppEnv          string
	AppPort         string
	DatabaseURL     string
	TokenIssuer     string
	TokenAudience   string
	TokenSecret     string
	AccessTokenTTL  time.Duration
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

func Load() (Config, error) {
	var cfg Config
	var errs []error

	cfg.AppEnv = strings.TrimSpace(os.Getenv("APP_ENV"))
	cfg.AppPort = strings.TrimSpace(os.Getenv("APP_PORT"))
	cfg.DatabaseURL = strings.TrimSpace(os.Getenv("DATABASE_URL"))
	cfg.TokenIssuer = strings.TrimSpace(os.Getenv("TOKEN_ISSUER"))
	cfg.TokenAudience = strings.TrimSpace(os.Getenv("TOKEN_AUDIENCE"))
	cfg.TokenSecret = os.Getenv("TOKEN_SECRET")

	if cfg.AppEnv == "" {
		errs = append(errs, errors.New("APP_ENV is required"))
	} else if !validAppEnv(cfg.AppEnv) {
		errs = append(errs, fmt.Errorf("APP_ENV must be one of %q, %q, %q", EnvLocal, EnvTest, EnvProduction))
	}

	if cfg.AppPort == "" {
		errs = append(errs, errors.New("APP_PORT is required"))
	} else if err := validatePort(cfg.AppPort); err != nil {
		errs = append(errs, fmt.Errorf("APP_PORT is invalid: %w", err))
	}

	if cfg.DatabaseURL == "" {
		errs = append(errs, errors.New("DATABASE_URL is required"))
	} else if err := validateDatabaseURL(cfg.DatabaseURL); err != nil {
		errs = append(errs, fmt.Errorf("DATABASE_URL is invalid: %w", err))
	}

	if cfg.TokenIssuer == "" {
		errs = append(errs, errors.New("TOKEN_ISSUER is required"))
	}
	if cfg.TokenAudience == "" {
		errs = append(errs, errors.New("TOKEN_AUDIENCE is required"))
	}
	if strings.TrimSpace(cfg.TokenSecret) == "" {
		errs = append(errs, errors.New("TOKEN_SECRET is required"))
	} else if cfg.AppEnv == EnvProduction && len(cfg.TokenSecret) < 32 {
		errs = append(errs, errors.New("TOKEN_SECRET must be at least 32 bytes in production"))
	}

	parseDuration := func(name string) time.Duration {
		value := strings.TrimSpace(os.Getenv(name))
		if value == "" {
			errs = append(errs, fmt.Errorf("%s is required", name))
			return 0
		}

		duration, err := time.ParseDuration(value)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s is invalid: %w", name, err))
			return 0
		}
		if duration <= 0 {
			errs = append(errs, fmt.Errorf("%s must be greater than zero", name))
			return 0
		}

		return duration
	}

	cfg.AccessTokenTTL = parseDuration("ACCESS_TOKEN_TTL")
	cfg.ReadTimeout = parseDuration("READ_TIMEOUT")
	cfg.WriteTimeout = parseDuration("WRITE_TIMEOUT")
	cfg.IdleTimeout = parseDuration("IDLE_TIMEOUT")
	cfg.ShutdownTimeout = parseDuration("SHUTDOWN_TIMEOUT")

	return cfg, errors.Join(errs...)
}

func validAppEnv(value string) bool {
	switch value {
	case EnvLocal, EnvTest, EnvProduction:
		return true
	default:
		return false
	}
}

func validatePort(value string) error {
	port, err := strconv.Atoi(value)
	if err != nil {
		return err
	}
	if port < 1 || port > 65535 {
		return fmt.Errorf("must be between 1 and 65535")
	}

	return nil
}

func validateDatabaseURL(value string) error {
	parsed, err := url.Parse(value)
	if err != nil {
		return err
	}
	if parsed.Scheme != "postgres" && parsed.Scheme != "postgresql" {
		return fmt.Errorf("scheme must be postgres or postgresql")
	}
	if parsed.Host == "" {
		return fmt.Errorf("host is required")
	}
	if parsed.Hostname() != "" && parsed.Port() != "" {
		if _, err := net.LookupPort("tcp", parsed.Port()); err != nil {
			return err
		}
	}
	if strings.TrimPrefix(parsed.Path, "/") == "" {
		return fmt.Errorf("database name is required")
	}

	return nil
}
