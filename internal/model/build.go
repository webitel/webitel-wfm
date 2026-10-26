package model

// ServiceName is the name the service registers under in discovery.
const ServiceName = "wfm"

// Build information. CI sets it through the cmd package (see cmd/run.go), which calls SetBuild.
var (
	Version        = "0.0.0"
	Commit         = "hash"
	CommitDate     = ""
	Branch         = "branch"
	BuildTimestamp = ""
)

// SetBuild overrides the build information with the values stamped at link time; empty values keep the defaults.
func SetBuild(version, commit, branch string) {
	if version != "" {
		Version = version
	}

	if commit != "" {
		Commit = commit
	}

	if branch != "" {
		Branch = branch
	}
}
