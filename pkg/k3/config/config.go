package config

import (
	"github.com/koding/multiconfig"
	"log-engine-sdk/pkg/k3/protocol"
	"strings"
	"sync"
)

type Config struct {
	ELK      ELK      `yaml:"elk" json:"elk" toml:"elk"`
	System   System   `yaml:"system" json:"system" toml:"system"`
	Http     Http     `yaml:"http" json:"http" toml:"http"`
	Consumer Consumer `yaml:"consumer" json:"consumer" toml:"consumer"`
	Watch    Watch    `yaml:"watch" json:"watch" toml:"watch"`
	Account  Account  `yaml:"account" json:"account"`
}

type ELK struct {
	Address        []string `yaml:"address" json:"addresses,omitempty" toml:"addresses"` // A list of Elasticsearch nodes to use.
	Username       string   `yaml:"username" json:"username,omitempty" toml:"username"`  // Username for HTTP Basic Authentication.
	Password       string   `yaml:"password" json:"password,omitempty" toml:"password"`  // Password for HTTP Basic Authentication.
	MaxChannelSize int      `yaml:"max_channel_size"`                                    // 最大管道
	MaxRetry       int      `yaml:"max_retry"`
	RetryInterval  int      `yaml:"retry_interval"`
	Timeout        int      `yaml:"timeout"`
}

type Watch struct {
	ReadPath        map[string][]string `yaml:"read_path" json:"read_path,omitempty" toml:"read_path"` // 要读取的日志文件路径
	IsUseSuffixDate bool                `yaml:"is_use_suffix_date" json:"is_use_suffix_date" toml:"is_use_suffix_date"`
	StateFilePath   string              `yaml:"state_file_path" json:"state_file_path,omitempty" toml:"state_file_path"`
	MaxReadCount    int                 `yaml:"max_read_count" json:"max_read_count"` // max_read_count
}

type System struct {
	PrintEnabled bool   `yaml:"print_enabled" json:"print_enabled,omitempty" toml:"print_enabled"`
	UseELK       bool   `yaml:"use_elk" json:"use_elk,omitempty" toml:"use_elk"`
	RootPath     string `yaml:"root_path" json:"root_path" toml:"root_path"`
	LogLevel     int    `yaml:"log_level"`
}

type Account struct {
	AccountId string `yaml:"account_id"`
	AppId     string `yaml:"app_id"`
	EventName string `yaml:"event_name"`
	EventId   string `yaml:"event_id"`
}

type Consumer struct {
	ConsumerLogChannelSize int  `yaml:"consumer_log_channel_size"` // 批量日志检查缓存列表时间间隔
	ConsumerBatchInterval  int  `yaml:"consumer_batch_interval"`   // 秒
	ConsumerBatchSize      int  `yaml:"consumer_batch_size"`       // 批量日志单次批量提交最大值
	ConsumerBatchCapacity  int  `yaml:"consumer_batch_capacity"`   // 批量日志缓存容量
	ConsumerBatchAutoFlush bool `yaml:"consumer_batch_auto_flush"` // 批量日志是否自动刷新
}

type Http struct {
	Port            int    `yaml:"port"`
	Host            string `yaml:"host"`
	ReadTimeout     int    `yaml:"read_timeout"`
	WriteTimeout    int    `yaml:"write_timeout"`
	IdleTimeout     int    `yaml:"idle_timeout"`
	ShutdownTimeout int    `yaml:"shutdown_timeout"`
	Enable          bool   `yaml:"enable"`
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
