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

const BeforeHardForkVersion = 0
//If height < NextHardForkHeight,run currentHardForkVersion
//else run NextHardForkVersion
const NextHardForkHeight = 400000
const NextHardForkVersion = 1

//If we are in the version=4,we should remember all the
//pre-version code and per-version height
//and run the corresponding login to get the same state
//as the main network
