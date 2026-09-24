package config

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"

	"github.com/fsnotify/fsnotify"
	"github.com/google/uuid"
	"github.com/spf13/pflag"

	"github.com/webitel/webitel-go-kit/appconfig"
	"github.com/webitel/webitel-go-kit/logging/wlog"
)

type Config struct {
	Service  Service            `mapstructure:"service"`
	Log      appconfig.Log      `mapstructure:"log"`
	Postgres appconfig.Postgres `mapstructure:"postgres"`
	Forecast appconfig.Postgres `mapstructure:"forecast"`
	Cache    Cache              `mapstructure:"cache"`
	Consul   appconfig.Consul   `mapstructure:"consul"`
	Pubsub   appconfig.Pubsub   `mapstructure:"pubsub"`
}

type Service struct {
	NodeID    string `mapstructure:"node_id"`
	Addr      string `mapstructure:"addr"`
	ProbeAddr string `mapstructure:"probe_addr"`
}

type Cache struct {
	Size int `mapstructure:"size"`
}

// LoadServerConfig loads the full configuration required by the gRPC server.
func LoadServerConfig() (*Config, error) {
	loader := appconfig.NewLoader(appconfig.Sections{
		Log:      true,
		Postgres: true,
		Consul:   true,
		Pubsub:   true,
	})
	loader.RegisterFlags(pflag.CommandLine)
	registerServiceFlags(pflag.CommandLine)
	pflag.Parse()

	cfg := &Config{}
	if err := loader.Load(pflag.CommandLine, cfg); err != nil {
		return nil, err
	}

	loader.Watch(func(e fsnotify.Event) {
		log := wlog.GlobalLogger()
		log.Info("config file changed", wlog.String("name", e.Name))

		newCfg := &Config{}
		if err := loader.Viper().Unmarshal(newCfg); err != nil {
			log.Error("config reload: unmarshal failed", wlog.Err(err))

			return
		}

		if err := newCfg.validate(); err != nil {
			log.Error("config reload: validation failed", wlog.Err(err))

			return
		}

		*cfg = *newCfg

		log.Info("config reloaded")
	})

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// LoadMigrateConfig loads the minimal configuration required by the migrate command.
func LoadMigrateConfig() (*Config, error) {
	loader := appconfig.NewLoader(appconfig.Sections{
		Log:      true,
		Postgres: true,
	})
	loader.RegisterFlags(pflag.CommandLine)
	pflag.Parse()

	cfg := &Config{}
	if err := loader.Load(pflag.CommandLine, cfg); err != nil {
		return nil, err
	}

	if cfg.Postgres.DSN == "" {
		return nil, errors.New("config: postgres.dsn is required")
	}

	return cfg, nil
}

func registerServiceFlags(fs *pflag.FlagSet) {
	fs.String("service.addr", "127.0.0.1:10031", "gRPC listen address")
	fs.String("service.probe_addr", "127.0.0.1:10033", "address serving /livez, /readyz and /healthz; empty disables them")
	fs.String("service.node_id", defaultNodeID(), "instance id registered in service discovery")
	fs.String("forecast.dsn", "", "PostgreSQL DSN for forecast calculation queries; defaults to postgres.dsn")
	fs.Int("cache.size", 1024, "in-memory cache capacity in bytes")
}

func defaultNodeID() string {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "wfm"
	}

	return hostname + "-" + uuid.NewString()
}

func (c *Config) validate() error {
	_, port, err := net.SplitHostPort(c.Service.Addr)
	if err != nil {
		return fmt.Errorf("parse service address: %w", err)
	}

	if _, err := strconv.Atoi(port); err != nil {
		return fmt.Errorf("parse service port: %w", err)
	}

	if c.Postgres.DSN == "" {
		return errors.New("config: postgres.dsn is required")
	}

	if c.Forecast.DSN == "" {
		c.Forecast.DSN = c.Postgres.DSN
	}

	if c.Log.Level == "" {
		c.Log.Level = "info"
	}

	return nil
}
