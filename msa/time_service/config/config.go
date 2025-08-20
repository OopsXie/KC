package config

import (
	"flag"
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v2"
)

type ServiceConfig struct {
	ServiceName       string `yaml:"serviceName"`
	ServiceID         string `yaml:"serviceId"`
	IpAddress         string `yaml:"ipAddress"`
	Port              int    `yaml:"port"`
	HeartbeatInterval int    `yaml:"heartbeatInterval"`
}

type RegistryConfig struct {
	Addresses []string `yaml:"addresses"`
}

type Config struct {
	Service  ServiceConfig  `yaml:"service"`
	Registry RegistryConfig `yaml:"registry"`
}

var Cfg Config

func GetConfigPathFromArgs() string {
	configPath := flag.String("config", "./config/time-service-1.yaml", "path to config file")
	flag.Parse()
	return *configPath
}

func LoadConfig(path string) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalf("[error] 读取配置失败: %v", err)
	}
	err = yaml.Unmarshal(data, &Cfg)
	if err != nil {
		log.Fatalf("[error] 解析配置失败: %v", err)
	}
}
