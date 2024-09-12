package config

import (
	"github.com/koding/multiconfig"
	"strings"
	"sync"
)

type Config struct {
	ELK    ELK
	System System
}

// TODO 需要考虑ELK的真实的配置需要哪些，目前只写了一些
type ELK struct {
	Addresses []string `yaml:"addresses"` // A list of Elasticsearch nodes to use.
	Username  string   `yaml:"username"`  // Username for HTTP Basic Authentication.
	Password  string   `yaml:"password"`  // Password for HTTP Basic Authentication.
	APIKey    string   `yaml:"api_key"`   // Base64-encoded token for authorization; if set, overrides username/password and service token.
}

type System struct {
	UseELK    bool   `yaml:"use_elk"`
	Version   string `yaml:"version"`
	ReadPath  string `yaml:"read_path"`
	AccountId string `yaml:"account_id"`
	AppId     string `yaml:"app_id"`
}

var (
	once         sync.Once
	GlobalConfig Config
)

func MustLoad(fpaths ...string) {
	once.Do(func() {
		var (
			loaders []multiconfig.Loader
			m       multiconfig.DefaultLoader
		)

		loaders = []multiconfig.Loader{
			&multiconfig.TagLoader{},
			&multiconfig.EnvironmentLoader{},
		}

		for _, fpath := range fpaths {
			if strings.HasSuffix(fpath, ".yaml") {
				loaders = append(loaders, &multiconfig.YAMLLoader{Path: fpath})
			}

			if strings.HasSuffix(fpath, ".json") {
				loaders = append(loaders, &multiconfig.JSONLoader{Path: fpath})
			}

			if strings.HasSuffix(fpath, ".toml") {
				loaders = append(loaders, &multiconfig.TOMLLoader{Path: fpath})
			}
		}

		m = multiconfig.DefaultLoader{
			Loader:    multiconfig.MultiLoader(loaders...),
			Validator: multiconfig.MultiValidator(&multiconfig.RequiredValidator{}),
		}

		m.MustLoad(&GlobalConfig)

	})
}
