package config

import (
	"github.com/koding/multiconfig"
	"log-engine-sdk/pkg/k3/protocol"
	"strings"
	"sync"
)

type Config struct {
	ELK    ELK    `yaml:"elk" json:"elk" toml:"elk"`
	System System `yaml:"system" json:"system" toml:"system"`
}

// TODO 需要考虑ELK的真实的配置需要哪些，目前只写了一些
type ELK struct {
	Address  []string `yaml:"address" json:"addresses,omitempty" toml:"addresses"` // A list of Elasticsearch nodes to use.
	Username string   `yaml:"username" json:"username,omitempty" toml:"username"`  // Username for HTTP Basic Authentication.
	Password string   `yaml:"password" json:"password,omitempty" toml:"password"`  // Password for HTTP Basic Authentication.
	ApiKey   string   `yaml:"api_key" json:"api_key,omitempty" toml:"api_key"`     // Base64-encoded token for authorization; if set, overrides username/password and service token.W0l

}

type System struct {
	PrintEnabled           bool     `yaml:"print_enabled" json:"print_enabled,omitempty" toml:"print_enabled"`
	UseELK                 bool     `yaml:"use_elk" json:"use_elk,omitempty" toml:"use_elk"`
	ReadPath               []string `yaml:"read_path" json:"read_path,omitempty" toml:"read_path"` // 要读取的日志文件路径
	StateFilePath          string   `yaml:"state_file_path" json:"state_file_path,omitempty" toml:"state_file_path"`
	RootPath               string   `yaml:"root_path" json:"root_path" toml:"root_path"`
	MaxReadCount           int      `yaml:"max_read_count"` // max_read_count
	ConsumerLogChannelSize int      `yaml:"consumer_log_channel_size"`
	ConsumerBatchSize      int      `yaml:"consumer_batch_size"`
	ConsumerBatchInterval  int      `yaml:"consumer_batch_interval"` // 秒
	ConsumerBatchCapacity  int      `yaml:"consumer_batch_capacity"`
}

var (
	once           sync.Once
	GlobalConfig   = new(Config)
	GlobalConsumer protocol.K3Consumer
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

		m.MustLoad(GlobalConfig)
	})
}
