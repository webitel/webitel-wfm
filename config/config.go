package config

import (
	"fmt"
	"net"
	"os"
	"strconv"

	"github.com/google/uuid"
)

type Logger struct {
	Level    string
	Type     string
	Endpoint string
}

type Service struct {
	NodeID        string
	Address       string
	MetricAddress string
}

type Database struct {
	DSN string
}

type Cache struct {
	Size    int64
	Type    string
	Address string
}

type Consul struct {
	Address string
}

type Pubsub struct {
	Address string
}

// Tracing contains the OpenTelemetry address and propagation config values.
type Tracing struct {
	Type        string
	Address     string
	Propagation string

	Sampler          string
	SamplerParam     float64
	SamplerRemoteURL string
}

// IsEnabled returns true if OTLP tracing is enabled (address set)
func (t Tracing) IsEnabled() bool {
	return t.Address != ""
}

type Config struct {
	Logger  Logger
	Tracing Tracing
	Service Service

	Database Database
	Cache    Cache

	Consul Consul
	Pubsub Pubsub
}

func New() *Config {
	id, err := uuid.NewUUID()
	if err != nil {
		panic(fmt.Errorf("generate UUID: %w", err))
	}

	hostname, err := os.Hostname()
	if err != nil {
		panic(fmt.Errorf("get hostname: %w", err))
	}

	return &Config{
		Logger: Logger{
			Level: "debug",
		},
		Service: Service{
			NodeID:        hostname + "-" + id.String(),
			Address:       "127.0.0.1:10031",
			MetricAddress: "127.0.0.1:10032",
		},
		Database: Database{
			DSN: "postgres://opensips:webitel@127.0.0.1:5432/webitel?application_name=wfm&sslmode=disable&connect_timeout=10",
		},
		Cache: Cache{
			Type: "inmemory",
		},
		Consul: Consul{
			Address: "127.0.0.1:8500",
		},
	}
}

func (c *Config) Load() error {
	_, port, err := net.SplitHostPort(c.Service.Address)
	if err != nil {
		return fmt.Errorf("parse service address: %w", err)
	}

	_, err = strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("parse service port: %w", err)
	}

	return nil
}
