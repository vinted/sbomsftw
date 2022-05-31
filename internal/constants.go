package internal

// Viper CLI keys
const (
	CLIKeyExclusions        = "cli-key-exclusions"
	CLIKeyOutput            = "cli-key-output"
	CLIKeyDTrackProjectName = "cli-key-dtrack-project-name"
)

// Output values for CLI --output switch
const (
	OutputValueStdout = "stdout"
	OutputValueDtrack = "dtrack"
)

// Viper ENV keys
const (
	EnvKeyGithubUsername = "GITHUB_USERNAME"
	EnvKeyGithubToken    = "GITHUB_TOKEN"
	EnvKeyDTrackURL      = "DEPENDENCY_TRACK_URL"
	EnvKeyDTrackToken    = "DEPENDENCY_TRACK_TOKEN"
)

const EnvPrefix = "SAC" // Software Asset Collector
