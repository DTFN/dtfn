package version

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

//profile variables
type conf struct {
	Develop    VersionData `yaml:"develop"`
	Staging    VersionData `yaml:"staging"`
	Production VersionData `yaml:"production"`
}

type VersionData struct {
	HeightString         string `yaml:"height"`
	VersionString        string `yaml:"version"`
	PPCAdmin             string `yaml:"ppcadmin"`
	BigGuy               string `yaml:"bigguy"`
	PPChainPrivateAdmin  string `yaml:"ppchainprivateadmin"`
	EvmErrHardForkHeight int64  `yaml:"evmerrhardforkheight"`
}

func ReadConfig(fileName string) (conf, error) {
	var c conf
	yamlFile, err := ioutil.ReadFile(fileName)
	if err != nil {
		fmt.Println(err.Error())
		return c, err
	}
	err = yaml.Unmarshal(yamlFile, &c)
	if err != nil {
		fmt.Println(err.Error())
	}
	return c, err
}

func LoadDevelopConfig(c conf) {
	HeightString = c.Develop.HeightString
	VersionString = c.Develop.VersionString
	PPChainAdmin = c.Develop.PPCAdmin
	Bigguy = c.Develop.BigGuy
}

func LoadStagingConfig(c conf) {
	HeightString = c.Staging.HeightString
	VersionString = c.Staging.VersionString
	PPChainAdmin = c.Staging.PPCAdmin
	Bigguy = c.Staging.BigGuy
}

func LoadProductionConfig(c conf) {
	HeightString = c.Production.HeightString
	VersionString = c.Production.VersionString
	PPChainAdmin = c.Production.PPCAdmin
	Bigguy = c.Production.BigGuy
}

func LoadDefaultConfig(c conf) {
	HeightString = ""
	VersionString = ""
	Bigguy = "0xb3d49259b486d04505b0b652ade74849c0b703c3"
	PPChainAdmin = "0xb3d49259b486d04505b0b652ade74849c0b703c3"
}