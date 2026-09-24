package model

// ServiceName is the name the service registers under in discovery.
const ServiceName = "wfm"

// Build information, set with -ldflags "-X github.com/webitel/webitel-wfm/internal/model.Version=...".
var (
	Version        = "0.0.0"
	Commit         = "hash"
	CommitDate     = ""
	Branch         = "branch"
	BuildTimestamp = ""
)
