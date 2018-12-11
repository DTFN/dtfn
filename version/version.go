package version

// Major version component of the current release
const Major = 0

// Minor version component of the current release
const Minor = 6

// Fix version component of the current release
const Fix = 0

var (
	// Version is the full version string
	Version = "0.6.0"

	// GitCommit is set with --ldflags "-X main.gitCommit=$(git rev-parse --short HEAD)"
	GitCommit string
)

func init() {
	if GitCommit != "" {
		Version += "-" + GitCommit
	}
}

const CurrentHardForkVersion = 0
const NextHardForkVersion = 1
const NextHardForkHeight = 210000