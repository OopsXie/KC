package cluster

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"msa/registry/config"
	"msa/registry/storage"
	"net/http"
	"sync"
	"time"
)

type ClusterManager struct {
	currentMaster    string
	mu               sync.RWMutex
	isActive         bool
	masterChangeChan chan bool // 主节点状态变化通知
}

var Manager = &ClusterManager{
	masterChangeChan: make(chan bool, 1),
}

type SyncRequest struct {
	Action   string                  `json:"action"`
	Instance storage.ServiceInstance `json:"instance"`
}

// getCurrentAddr 获取当前节点地址
func (cm *ClusterManager) getCurrentAddr() string {
	return config.GetCurrentNodeAddr()
}

// Start 启动集群管理器
func (cm *ClusterManager) Start() {
	cm.mu.Lock()
	cm.isActive = true

	currentAddr := cm.getCurrentAddr()

	// 首先尝试发现当前的活跃主节点
	activeMaster := cm.discoverActiveMaster()
	if activeMaster != "" {
		cm.currentMaster = activeMaster
		log.Printf("[cluster] 发现活跃主节点: %s", cm.currentMaster)
	} else {
		// 没有活跃主节点，使用配置文件中的默认主节点
		cm.currentMaster = config.GetMasterAddr()
		log.Printf("[cluster] 没有发现活跃主节点，使用默认主节点: %s", cm.currentMaster)
	}

	// 检查当前节点是否为主节点，如果是则发送通知
	if cm.currentMaster == currentAddr {
		log.Printf("[cluster] 集群管理器启动，当前节点为主节点: %s", cm.currentMaster)
		// 发送主节点状态通知
		select {
		case cm.masterChangeChan <- true:
		default:
		}
	} else {
		log.Printf("[cluster] 集群管理器启动，当前主节点: %s, 本节点: %s", cm.currentMaster, currentAddr)
		// 从节点启动时，尝试从主节点同步数据
		go cm.syncDataFromMaster()
	}

	cm.mu.Unlock()

	// 启动主节点健康检查
	go cm.monitorMaster()
} // IsMaster 判断当前节点是否为主节点
func (cm *ClusterManager) IsMaster() bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	currentAddr := cm.getCurrentAddr()
	return cm.currentMaster == currentAddr
}

// GetMaster 获取当前主节点地址
func (cm *ClusterManager) GetMaster() string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.currentMaster
}

// ForwardToMaster 将请求转发到主节点
func (cm *ClusterManager) ForwardToMaster(action string, instance storage.ServiceInstance) (*http.Response, error) {
	masterAddr := cm.GetMaster()
	currentAddr := cm.getCurrentAddr()

	if masterAddr == currentAddr {
		return nil, fmt.Errorf("当前节点就是主节点")
	}

	var endpoint string
	switch action {
	case "register":
		endpoint = "/api/register"
	case "unregister":
		endpoint = "/api/unregister"
	case "heartbeat":
		endpoint = "/api/heartbeat"
	default:
		return nil, fmt.Errorf("不支持的操作: %s", action)
	}

	jsonData, err := json.Marshal(instance)
	if err != nil {
		return nil, fmt.Errorf("序列化数据失败: %v", err)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("POST", masterAddr+endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		// 主节点可能故障，尝试故障转移
		log.Printf("[cluster] 转发到主节点失败: %v", err)
		cm.handleMasterFailure()
		return nil, fmt.Errorf("转发到主节点失败: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("主节点响应异常: %d", resp.StatusCode)
	}

	return resp, nil
}

// SyncToSlaves 主节点同步数据到从节点
func (cm *ClusterManager) SyncToSlaves(action string, instance storage.ServiceInstance) {
	if !cm.IsMaster() {
		return
	}

	slaveAddrs := config.GetSlaveAddrs()
	for _, slaveAddr := range slaveAddrs {
		go func(addr string) {
			err := cm.syncToSlave(addr, action, instance)
			if err != nil {
				log.Printf("[cluster] 同步到从节点 %s 失败: %v", addr, err)
			}
		}(slaveAddr)
	}
}

// syncToSlave 同步数据到单个从节点
func (cm *ClusterManager) syncToSlave(slaveAddr, action string, instance storage.ServiceInstance) error {
	syncReq := SyncRequest{
		Action:   action,
		Instance: instance,
	}

	jsonData, err := json.Marshal(syncReq)
	if err != nil {
		return fmt.Errorf("序列化同步数据失败: %v", err)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("POST", slaveAddr+"/api/internal/sync", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("创建同步请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("发送同步请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("从节点响应异常: %d", resp.StatusCode)
	}

	return nil
}

// HandleSlaveSync 处理来自主节点的同步请求
func (cm *ClusterManager) HandleSlaveSync(action string, instance storage.ServiceInstance) bool {
	if cm.IsMaster() {
		return false // 主节点不处理同步请求
	}

	switch action {
	case "register":
		return storage.SaveInstanceForSync(instance)
	case "unregister":
		return storage.RemoveInstanceInternal(instance.ServiceID)
	case "heartbeat":
		// 对于心跳同步，使用主节点传过来的时间信息
		return storage.UpdateHeartbeatWithTime(instance.ServiceID, instance.LastHeartbeat, instance.LastHeartbeatGMTTime)
	default:
		log.Printf("[cluster] 未知的同步操作: %s", action)
		return false
	}
}

// monitorMaster 监控主节点健康状态
func (cm *ClusterManager) monitorMaster() {
	checkInterval := time.Duration(config.GetClusterCheckIntervalSeconds()) * time.Second
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for range ticker.C {
		if !cm.isActive {
			return
		}

		masterAddr := cm.GetMaster()
		currentAddr := cm.getCurrentAddr()

		// 如果当前节点就是主节点，不需要检查
		if masterAddr == currentAddr {
			continue
		}

		// 检查主节点健康状态
		if !cm.checkMasterHealth(masterAddr) {
			log.Printf("[cluster] 主节点 %s 不可用，尝试故障转移", masterAddr)
			cm.handleMasterFailure()
		}
	}
}

// checkMasterHealth 检查主节点健康状态
func (cm *ClusterManager) checkMasterHealth(masterAddr string) bool {
	client := &http.Client{Timeout: 3 * time.Second}
	// 使用专门的健康检查端点
	resp, err := client.Get(masterAddr + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// 只要能成功响应就认为节点健康
	return resp.StatusCode == http.StatusOK
}

// handleMasterFailure 处理主节点故障
func (cm *ClusterManager) handleMasterFailure() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	clusterNodes := config.Cfg.Registry.Cluster
	currentMasterIndex := -1
	currentAddr := cm.getCurrentAddr()

	// 找到当前主节点在集群中的位置
	for i, addr := range clusterNodes {
		if addr == cm.currentMaster {
			currentMasterIndex = i
			break
		}
	}

	// 从下一个节点开始寻找新的主节点
	for i := currentMasterIndex + 1; i < len(clusterNodes); i++ {
		if clusterNodes[i] == currentAddr {
			// 如果当前节点就是下一个候选节点，直接成为主节点
			oldMaster := cm.currentMaster
			cm.currentMaster = clusterNodes[i]
			log.Printf("[cluster] 主节点从 %s 切换到 %s (当前节点)", oldMaster, cm.currentMaster)
			// 通知主节点状态变化
			select {
			case cm.masterChangeChan <- true:
			default:
			}
			// 成为新主节点时，从旧主节点同步轮询状态（如果可能）
			if oldMaster != cm.currentMaster && cm.checkMasterHealth(oldMaster) {
				go cm.syncLoadBalanceStateFromMaster(oldMaster)
			}
			return
		} else if cm.checkMasterHealth(clusterNodes[i]) {
			oldMaster := cm.currentMaster
			cm.currentMaster = clusterNodes[i]
			log.Printf("[cluster] 主节点从 %s 切换到 %s", oldMaster, cm.currentMaster)
			// 通知主节点状态变化
			select {
			case cm.masterChangeChan <- false:
			default:
			}
			// 切换到新主节点时，同步轮询状态
			go cm.syncLoadBalanceStateFromMaster(cm.currentMaster)
			return
		}
	}

	// 如果没有找到下一个可用节点，从头开始查找
	for i := 0; i < currentMasterIndex; i++ {
		if clusterNodes[i] == currentAddr {
			// 如果当前节点就是候选节点，直接成为主节点
			oldMaster := cm.currentMaster
			cm.currentMaster = clusterNodes[i]
			log.Printf("[cluster] 主节点从 %s 切换到 %s (当前节点)", oldMaster, cm.currentMaster)
			// 通知主节点状态变化
			select {
			case cm.masterChangeChan <- true:
			default:
			}
			// 成为新主节点时，从旧主节点同步轮询状态（如果可能）
			if oldMaster != cm.currentMaster && cm.checkMasterHealth(oldMaster) {
				go cm.syncLoadBalanceStateFromMaster(oldMaster)
			}
			return
		} else if cm.checkMasterHealth(clusterNodes[i]) {
			oldMaster := cm.currentMaster
			cm.currentMaster = clusterNodes[i]
			log.Printf("[cluster] 主节点从 %s 切换到 %s", oldMaster, cm.currentMaster)
			// 通知主节点状态变化
			select {
			case cm.masterChangeChan <- false:
			default:
			}
			// 切换到新主节点时，同步轮询状态
			go cm.syncLoadBalanceStateFromMaster(cm.currentMaster)
			return
		}
	}

	log.Printf("[cluster] 没有找到可用的主节点")
}

// Stop 停止集群管理器
func (cm *ClusterManager) Stop() {
	cm.mu.Lock()
	cm.isActive = false
	cm.mu.Unlock()
}

// GetMasterChangeNotification 获取主节点状态变化通知
func (cm *ClusterManager) GetMasterChangeNotification() chan bool {
	return cm.masterChangeChan
}

// discoverActiveMaster 发现当前活跃的主节点
func (cm *ClusterManager) discoverActiveMaster() string {
	clusterNodes := config.Cfg.Registry.Cluster
	currentAddr := cm.getCurrentAddr()

	// 先检查其他节点，不检查自己
	for _, addr := range clusterNodes {
		if addr != currentAddr && cm.checkMasterHealth(addr) {
			log.Printf("[cluster] 发现活跃节点: %s", addr)
			return addr
		}
	}

	// 如果其他节点都不可用，再检查自己
	if cm.checkMasterHealth(currentAddr) {
		log.Printf("[cluster] 其他节点均不可用，当前节点成为主节点: %s", currentAddr)
		return currentAddr
	}

	return ""
}

// syncDataFromMaster 从主节点同步所有服务数据
func (cm *ClusterManager) syncDataFromMaster() {
	// 等待1秒，确保集群状态稳定
	time.Sleep(1 * time.Second)

	masterAddr := cm.GetMaster()
	currentAddr := cm.getCurrentAddr()

	if masterAddr == currentAddr {
		log.Printf("[cluster] 当前节点就是主节点，无需同步数据")
		return
	}

	log.Printf("[cluster] 开始从主节点 %s 同步服务数据", masterAddr) // 调用主节点的discovery接口获取所有服务
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(masterAddr + "/api/discovery")
	if err != nil {
		log.Printf("[cluster] 从主节点同步数据失败: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[cluster] 主节点返回异常状态: %d", resp.StatusCode)
		return
	}

	// 解析响应
	var response struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			TotalCount int `json:"totalCount"`
			Instances  []struct {
				ServiceName          string `json:"serviceName"`
				ServiceID            string `json:"serviceId"`
				IPAddress            string `json:"ipAddress"`
				Port                 int    `json:"port"`
				RegistrationTime     int64  `json:"registrationTime"`
				LastHeartbeatTime    int64  `json:"lastHeartbeatTime"`
				RegistrationGMTTime  string `json:"registrationGMTTime"`
				LastHeartbeatGMTTime string `json:"lastHeartbeatGMTTime"`
			} `json:"instances"`
		} `json:"data"`
	}

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		log.Printf("[cluster] 解析主节点服务数据失败: %v", err)
		return
	}

	if response.Code != 200 {
		log.Printf("[cluster] 主节点返回错误: %s", response.Msg)
		return
	}

	// 将服务数据保存到本地
	syncCount := 0
	for _, instance := range response.Data.Instances {
		serviceInstance := storage.ServiceInstance{
			ServiceName:          instance.ServiceName,
			ServiceID:            instance.ServiceID,
			IPAddress:            instance.IPAddress,
			Port:                 instance.Port,
			RegisteredAt:         instance.RegistrationTime,
			LastHeartbeat:        instance.LastHeartbeatTime,
			RegisteredGMTTime:    instance.RegistrationGMTTime,
			LastHeartbeatGMTTime: instance.LastHeartbeatGMTTime,
		}

		// 检查从主节点获取的数据是否有效
		if serviceInstance.RegisteredAt <= 0 || serviceInstance.LastHeartbeat <= 0 {
			log.Printf("[cluster] 警告: 从主节点同步的实例 %s 时间戳无效 (注册时间=%d, 心跳时间=%d)，跳过同步",
				serviceInstance.ServiceID, serviceInstance.RegisteredAt, serviceInstance.LastHeartbeat)
			continue
		}

		success := storage.SaveInstanceForSync(serviceInstance)
		if success {
			syncCount++
		}
	}

	log.Printf("[cluster] 从主节点同步完成，共同步 %d 个服务实例", syncCount)

	// 同步轮询状态
	cm.syncLoadBalanceStateFromMaster(masterAddr)
}

// TriggerMasterHealthCheck 触发主节点健康检查（用于故障快速检测）
func (cm *ClusterManager) TriggerMasterHealthCheck() {
	if !cm.isActive {
		return
	}

	masterAddr := cm.GetMaster()
	currentAddr := cm.getCurrentAddr()

	// 如果当前节点就是主节点，不需要检查
	if masterAddr == currentAddr {
		return
	}

	// 检查主节点健康状态
	if !cm.checkMasterHealth(masterAddr) {
		log.Printf("[cluster] 触发的健康检查发现主节点 %s 不可用，执行故障转移", masterAddr)
		cm.handleMasterFailure()
	}
}

// syncLoadBalanceStateFromMaster 从主节点同步轮询状态
func (cm *ClusterManager) syncLoadBalanceStateFromMaster(masterAddr string) {
	currentAddr := cm.getCurrentAddr()

	if masterAddr == currentAddr {
		log.Printf("[cluster] 当前节点就是主节点，无需同步轮询状态")
		return
	}

	log.Printf("[cluster] 开始从主节点 %s 同步轮询状态", masterAddr)

	// 调用主节点的轮询状态接口
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(masterAddr + "/api/internal/loadbalance")
	if err != nil {
		log.Printf("[cluster] 从主节点同步轮询状态失败: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[cluster] 主节点返回轮询状态异常状态: %d", resp.StatusCode)
		return
	}

	// 解析响应
	var response struct {
		Code int            `json:"code"`
		Msg  string         `json:"msg"`
		Data map[string]int `json:"data"`
	}

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		log.Printf("[cluster] 解析主节点轮询状态数据失败: %v", err)
		return
	}

	if response.Code != 200 {
		log.Printf("[cluster] 主节点返回轮询状态错误: %s", response.Msg)
		return
	}

	// 同步轮询状态到本地
	if response.Data != nil {
		storage.SyncLoadBalanceStateFromMaster(response.Data)
		log.Printf("[cluster] 轮询状态同步完成，共同步 %d 个服务的轮询状态", len(response.Data))
	} else {
		log.Printf("[cluster] 主节点返回空的轮询状态数据")
	}
}
