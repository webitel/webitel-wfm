package config

import (
	"strings"
	"testing"
	"time"

	"github.com/spf13/pflag"
)

func serverFlags(t *testing.T, args ...string) *pflag.FlagSet {
	t.Helper()

	fs := pflag.NewFlagSet("wfm", pflag.ContinueOnError)
	registerServiceFlags(fs)

	if err := fs.Parse(args); err != nil {
		t.Fatal(err)
	}

	return fs
}

func TestNodeIDSurvivesARestart(t *testing.T) {
	first, _ := serverFlags(t).GetString("service.node_id")
	second, _ := serverFlags(t).GetString("service.node_id")

	if first == "" || first != second || !strings.HasPrefix(first, "wfm@") {
		t.Fatalf("node ids %q and %q, want one wfm@<hostname>", first, second)
	}
}

func TestLeaderElectionDefaults(t *testing.T) {
	fs := serverFlags(t)

	for flag, want := range map[string]time.Duration{
		"leader_election.session_ttl":      15 * time.Second,
		"leader_election.lock_delay":       time.Second,
		"leader_election.retry_interval":   10 * time.Second,
		"leader_election.error_cooldown":   5 * time.Second,
		"leader_election.monitor_interval": 5 * time.Second,
	} {
		if got, err := fs.GetDuration(flag); err != nil || got != want {
			t.Errorf("%s = %s (%v), want %s", flag, got, err, want)
		}
	}
}

func TestValidate(t *testing.T) {
	valid := func() *Config {
		return &Config{
			Service: Service{NodeID: "wfm@host", Addr: "127.0.0.1:10031"},
			LeaderElection: LeaderElection{
				SessionTTL:      15 * time.Second,
				LockDelay:       time.Second,
				RetryInterval:   10 * time.Second,
				ErrorCooldown:   5 * time.Second,
				MonitorInterval: 5 * time.Second,
			},
		}
	}

	cfg := valid()
	cfg.Postgres.DSN = "dsn"

	if err := cfg.validate(); err != nil {
		t.Fatalf("valid config refused: %v", err)
	}

	for name, breakIt := range map[string]func(*Config){
		"no node id":         func(c *Config) { c.Service.NodeID = "" },
		"ttl below consul's": func(c *Config) { c.LeaderElection.SessionTTL = 9 * time.Second },
		"ttl above consul's": func(c *Config) { c.LeaderElection.SessionTTL = 25 * time.Hour },
		"no lock delay":      func(c *Config) { c.LeaderElection.LockDelay = 0 },
	} {
		cfg := valid()
		cfg.Postgres.DSN = "dsn"
		breakIt(cfg)

		if err := cfg.validate(); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}
