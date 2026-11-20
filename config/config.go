package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/pflag"

	"github.com/webitel/webitel-go-kit/appconfig"
	"github.com/webitel/webitel-go-kit/infra/discovery"

	"github.com/webitel/webitel-wfm/internal/model"
)

type Config struct {
	Service  Service            `mapstructure:"service"`
	Log      appconfig.Log      `mapstructure:"log"`
	Postgres appconfig.Postgres `mapstructure:"postgres"`
	Forecast appconfig.Postgres `mapstructure:"forecast"`
	Consul   appconfig.Consul   `mapstructure:"consul"`
	Pubsub   appconfig.Pubsub   `mapstructure:"pubsub"`

	LeaderElection LeaderElection `mapstructure:"leader_election"`
}

type Service struct {
	NodeID    string `mapstructure:"node_id"`
	Addr      string `mapstructure:"addr"`
	ProbeAddr string `mapstructure:"probe_addr"`
}

// LeaderElection tunes the consul session that elects the instance running background jobs.
type LeaderElection struct {
	SessionTTL      time.Duration `mapstructure:"session_ttl"`
	LockDelay       time.Duration `mapstructure:"lock_delay"`
	RetryInterval   time.Duration `mapstructure:"retry_interval"`
	ErrorCooldown   time.Duration `mapstructure:"error_cooldown"`
	MonitorInterval time.Duration `mapstructure:"monitor_interval"`
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
		log := slog.Default()
		log.Info("config file changed", slog.String("name", e.Name))

		newCfg := &Config{}
		if err := loader.Viper().Unmarshal(newCfg); err != nil {
			log.Error("config reload: unmarshal failed", slog.Any("error", err))

			return
		}

		if err := newCfg.validate(); err != nil {
			log.Error("config reload: validation failed", slog.Any("error", err))

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
	fs.String("service.node_id", discovery.GenerateInstanceID(model.ServiceName), "instance id, unique per instance and stable across restarts: it owns the leader key and job leases")
	fs.String("forecast.dsn", "", "PostgreSQL DSN for forecast calculation queries; defaults to postgres.dsn")
	fs.Duration("leader_election.session_ttl", 15*time.Second, "consul leader session TTL (10s..24h)")
	fs.Duration("leader_election.lock_delay", time.Second, "delay before the leader key can be taken after a session dies")
	fs.Duration("leader_election.retry_interval", 10*time.Second, "how often a standby instance tries to become leader")
	fs.Duration("leader_election.error_cooldown", 5*time.Second, "pause after a failed consul call or leader term")
	fs.Duration("leader_election.monitor_interval", 5*time.Second, "how often the leader checks that it still holds the key")
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

	if c.Service.NodeID == "" {
		return errors.New("config: service.node_id is required")
	}

	// Consul refuses a session TTL outside this range.
	if ttl := c.LeaderElection.SessionTTL; ttl < 10*time.Second || ttl > 24*time.Hour {
		return errors.New("config: leader_election.session_ttl must be within 10s..24h")
	}

	// Zero would silently fall back to the consul default of 15s.
	if c.LeaderElection.LockDelay <= 0 {
		return errors.New("config: leader_election.lock_delay must be positive")
	}

	return nil
}
