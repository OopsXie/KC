package config

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v2"
)

type RegistryConfig struct {
	Port       int             `yaml:"port"`
	InstanceID string          `yaml:"instanceId"`
	Cluster    []string        `yaml:"cluster"`   // 集群节点列表，按优先级排序
	Heartbeat  HeartbeatConfig `yaml:"heartbeat"` // 心跳相关配置
}

type HeartbeatConfig struct {
	TimeoutSeconds       int `yaml:"timeoutSeconds"`       // 心跳超时时间（秒），默认60秒
	CleanupInterval      int `yaml:"cleanupInterval"`      // 清理任务检查间隔（秒），默认10秒
	ClusterCheckInterval int `yaml:"clusterCheckInterval"` // 集群状态检查间隔（秒），默认10秒
}

type Config struct {
	Registry RegistryConfig `yaml:"registry"`
}

var Cfg Config

func GetConfigPathFromArgs() string {
	configPath := flag.String("config", "./config/registry-docker-1.yaml", "path to config file")
	flag.Parse()
	return *configPath
}

func LoadConfig(path string) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalf("读取配置失败: %v", err)
	}
	err = yaml.Unmarshal(data, &Cfg)
	if err != nil {
		log.Fatalf("解析配置失败: %v", err)
	}

	// 设置心跳配置默认值
	if Cfg.Registry.Heartbeat.TimeoutSeconds == 0 {
		Cfg.Registry.Heartbeat.TimeoutSeconds = 60 // 默认60秒超时
	}
	if Cfg.Registry.Heartbeat.CleanupInterval == 0 {
		Cfg.Registry.Heartbeat.CleanupInterval = 10 // 默认10秒检查间隔
	}
	if Cfg.Registry.Heartbeat.ClusterCheckInterval == 0 {
		Cfg.Registry.Heartbeat.ClusterCheckInterval = 10 // 默认10秒集群检查间隔
	}
}

// GetCurrentNodeAddr 获取当前节点地址
func GetCurrentNodeAddr() string {
	// 根据配置的instanceId和端口来匹配集群中的节点
	for _, addr := range Cfg.Registry.Cluster {
		// 从集群配置中找到与当前端口匹配的节点
		expectedAddr := fmt.Sprintf("http://xzh-%s:%d", Cfg.Registry.InstanceID, Cfg.Registry.Port)
		if addr == expectedAddr {
			return addr
		}
	}

	// 如果找不到匹配的节点，返回第一个节点（作为fallback）
	if len(Cfg.Registry.Cluster) > 0 {
		return Cfg.Registry.Cluster[0]
	}

	// 最后的fallback，使用instanceId和端口构造地址
	return fmt.Sprintf("http://xzh-%s:%d", Cfg.Registry.InstanceID, Cfg.Registry.Port)
}

// IsMaster 判断当前节点是否为主节点
func IsMaster() bool {
	currentAddr := GetCurrentNodeAddr()
	return len(Cfg.Registry.Cluster) > 0 && Cfg.Registry.Cluster[0] == currentAddr
}

// GetMasterAddr 获取主节点地址
func GetMasterAddr() string {
	if len(Cfg.Registry.Cluster) > 0 {
		return Cfg.Registry.Cluster[0]
	}
	return ""
}

// GetHeartbeatTimeoutSeconds 获取心跳超时时间（秒）
func GetHeartbeatTimeoutSeconds() int {
	return Cfg.Registry.Heartbeat.TimeoutSeconds
}

// GetCleanupIntervalSeconds 获取清理任务检查间隔（秒）
func GetCleanupIntervalSeconds() int {
	return Cfg.Registry.Heartbeat.CleanupInterval
}

// GetClusterCheckIntervalSeconds 获取集群状态检查间隔（秒）
func GetClusterCheckIntervalSeconds() int {
	return Cfg.Registry.Heartbeat.ClusterCheckInterval
}

// GetSlaveAddrs 获取除当前节点外的所有集群节点地址
func GetSlaveAddrs() []string {
	if len(Cfg.Registry.Cluster) <= 1 {
		return []string{}
	}

	currentAddr := GetCurrentNodeAddr()
	var slaveAddrs []string

	for _, addr := range Cfg.Registry.Cluster {
		if addr != currentAddr {
			slaveAddrs = append(slaveAddrs, addr)
		}
	}

	return slaveAddrs
}
