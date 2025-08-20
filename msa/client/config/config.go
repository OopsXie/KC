package config

import (
	"flag"
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v3"
)

type ClientConfig struct {
	ServiceName       string `yaml:"serviceName"`
	ServiceID         string `yaml:"serviceId"`
	IpAddress         string `yaml:"ipAddress"`
	Port              int    `yaml:"port"`
	HeartbeatInterval int    `yaml:"heartbeatInterval"`
	LoggingInterval   int    `yaml:"loggingInterval"`
}

type RegistryConfig struct {
	Addresses []string `yaml:"addresses"` // 支持多个地址
}

type LoggingConfig struct {
	BaseURL string `yaml:"baseURL"` // 日志服务基础URL
}

type Config struct {
	Client   ClientConfig   `yaml:"client"`
	Registry RegistryConfig `yaml:"registry"`
	Logging  LoggingConfig  `yaml:"logging"`
}

var Cfg Config

func GetConfigPathFromArgs() string {
	configPath := flag.String("config", "./config/client-2.yaml", "path to config file")
	flag.Parse()
	return *configPath
}

func LoadConfig(path string) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalf("[error] failed to read config file: %v", err)
	}
	if err := yaml.Unmarshal(data, &Cfg); err != nil {
		log.Fatalf("[error] failed to unmarshal config: %v", err)
	}
}
