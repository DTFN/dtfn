package version

import (
	"strconv"
	"strings"
)

// Major version component of the current release
const Major = 0

// Minor version component of the current release
const Minor = 6

// Fix version component of the current release
const Fix = 0

var (
	// Version is the full version string
	Version = "1.5.0"

	// GitCommit is set with --ldflags "-X main.gitCommit=$(git rev-parse --short HEAD)"
	GitCommit string

	HeightArray []int64

	VersionArray []int64

	PPChainAdmin string

	PPChainPrivateAdmin string

	Bigguy string

	EvmErrHardForkHeight int64

	HeightString string

	VersionString string
)

func init() {
	if GitCommit != "" {
		Version += "-" + GitCommit
	}
}

func InitConfig() {
	if GitCommit != "" {
		Version += "-" + GitCommit
	}
	heightStrArray := strings.Split(HeightString, ",")
	versionStrArray := strings.Split(VersionString, ",")

	HeightArray = make([]int64, len(heightStrArray))
	VersionArray = make([]int64, len(versionStrArray))
	for i := 0; i < len(heightStrArray); i++ {
		HeightArray[i], _ = strconv.ParseInt(heightStrArray[i], 10, 64)
		VersionArray[i], _ = strconv.ParseInt(versionStrArray[i], 10, 64)
	}
}
