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
)

func init() {
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

const BeforeHardForkVersion = 0

//If height < NextHardForkHeight,run currentHardForkVersion
//else run NextHardForkVersion
//const PreHardForkHeight = 85000
//const PreHardForeVersion = 2
//const NextHardForkHeight = 1300000
//const NextHardForkVersion = 3

const HeightString = "85000,1300000,3623000"
const VersionString = "2,3,4"

const PPChainAdmin = "0xb3d49259b486d04505b0b652ade74849c0b703c3"

const Bigguy = "0x63859f245ba7c3c407a603a6007490b217b3ec5d"

//If we are in the version=4,we should remember all the
//pre-version code and per-version height
//and run the corresponding login to get the same state
//as the main network
//such
/*
const BeforeFirstHardForkVersion 0
const FirstHardForkHeight  400
const FirstHardForkVersion 1
const SecondHardForkHeight 800
const SecondHardForkVersion 2
const ThirdHardForkHeight  1200
const ThirdHardForkVersion  3
const NextHardForkHeight 1600
const NextHardForkVersion 4
*/

//upgrade node
/*
 The beginning version is `1`
 and we are in the height of 85000 hardforked first time
 which select 7 nodes.
 FirstHardForkVersion 2  FirstHardForkHeight 85000 From version 1 to 2


 We want to fix the AddBalanceBug at version 3 at height 1000000
 from `ws.state.AddBalance(beneficiary, bonusAverage)`
 to `ws.state.AddBalance(beneficiary, bonusSpecify)`
 SecondHardForkVersion 3 SecondHardForkHeight 1000000 From version 2 to 3
*/
