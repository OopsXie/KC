package storage

import (
	"log"
	"msa/registry/config"
	"net/http"
	"sync"
	"time"
)

type ServiceInstance struct {
	ServiceName string `json:"serviceName"`
	ServiceID   string `json:"serviceId"`
	IPAddress   string `json:"ipAddress"`
	Port        int    `json:"port"`

	RegisteredAt  int64 `json:"registeredAt,omitempty"`
	LastHeartbeat int64 `json:"lastHeartbeat,omitempty"`

	RegisteredGMTTime    string `json:"registeredGMTTime,omitempty"`
	LastHeartbeatGMTTime string `json:"lastHeartbeatGMTTime,omitempty"`
}

var (
	serviceMap = make(map[string][]ServiceInstance)
	mapLock    = sync.RWMutex{}
	rrIndexMap = make(map[string]int) // 每个服务的轮询索引
)

// SaveInstance 添加服务实例（注册） - 带集群管理
func SaveInstance(ins ServiceInstance) bool {
	clusterMgr := getClusterManager()

	if clusterMgr != nil && clusterMgr.IsMaster() {
		// 主节点：直接保存并同步到从节点
		log.Printf("[storage] 当前节点为主节点，直接保存服务实例: %s", ins.ServiceID)
		success := SaveInstanceInternal(ins)
		if success {
			clusterMgr.SyncToSlaves("register", ins)
		}
		return success
	} else if clusterMgr != nil {
		// 从节点：转发到主节点
		log.Printf("[storage] 当前节点为从节点，转发注册请求到主节点: %s", ins.ServiceID)
		resp, err := clusterMgr.ForwardToMaster("register", ins)
		if resp != nil {
			resp.Body.Close()
		}
		if err != nil {
			log.Printf("[storage] 转发到主节点失败: %v", err)
			return false
		}
		return true
	} else {
		// 无集群管理，直接保存
		log.Printf("[storage] 无集群管理，直接保存服务实例: %s", ins.ServiceID)
		return SaveInstanceInternal(ins)
	}
}

// SaveInstanceInternal 内部保存方法，不触发同步
func SaveInstanceInternal(ins ServiceInstance) bool {
	mapLock.Lock()
	defer mapLock.Unlock()

	// 检查ServiceID是否已存在
	for serviceName, list := range serviceMap {
		for i, exist := range list {
			if exist.ServiceID == ins.ServiceID {
				// 如果存在，检查是否是心跳更新
				// 1. 如果只有心跳字段有值（其他字段为空），认为是简单心跳更新
				if ins.ServiceName == "" && ins.IPAddress == "" && ins.Port == 0 &&
					(ins.LastHeartbeat > 0 || ins.LastHeartbeatGMTTime != "") {
					// 更新心跳时间（使用传入的时间戳，如果没有则使用当前时间）
					if ins.LastHeartbeat > 0 {
						serviceMap[serviceName][i].LastHeartbeat = ins.LastHeartbeat
					} else {
						serviceMap[serviceName][i].LastHeartbeat = time.Now().Unix()
					}

					if ins.LastHeartbeatGMTTime != "" {
						serviceMap[serviceName][i].LastHeartbeatGMTTime = ins.LastHeartbeatGMTTime
					} else {
						serviceMap[serviceName][i].LastHeartbeatGMTTime = time.Now().UTC().Format("2006-01-02 15:04:05")
					}
					return true
				}

				// 2. 如果包含完整服务信息且服务信息匹配，认为是完整心跳更新
				if ins.ServiceName == exist.ServiceName && ins.IPAddress == exist.IPAddress && ins.Port == exist.Port {
					// 只更新心跳时间，保持注册时间不变
					if ins.LastHeartbeat > 0 {
						serviceMap[serviceName][i].LastHeartbeat = ins.LastHeartbeat
					} else {
						serviceMap[serviceName][i].LastHeartbeat = time.Now().Unix()
					}

					if ins.LastHeartbeatGMTTime != "" {
						serviceMap[serviceName][i].LastHeartbeatGMTTime = ins.LastHeartbeatGMTTime
					} else {
						serviceMap[serviceName][i].LastHeartbeatGMTTime = time.Now().UTC().Format("2006-01-02 15:04:05")
					}
					return true
				}

				return false // ServiceID已存在，但服务信息不匹配
			}
		}
	}

	// 设置 RegisteredAt 和 LastHeartbeat 为当前时间戳（只在首次注册时设置）
	currentTime := time.Now()

	// 如果 RegisteredAt 为 0，说明是首次注册，设置注册时间
	if ins.RegisteredAt == 0 {
		ins.RegisteredAt = currentTime.UTC().Unix()
		ins.RegisteredGMTTime = currentTime.UTC().Format("2006-01-02 15:04:05")
	}

	// 设置心跳时间
	ins.LastHeartbeat = currentTime.UTC().Unix()
	ins.LastHeartbeatGMTTime = currentTime.UTC().Format("2006-01-02 15:04:05")

	// ServiceID不存在，添加新实例
	serviceMap[ins.ServiceName] = append(serviceMap[ins.ServiceName], ins)
	return true // 添加成功
}

// SaveInstanceForSync 专门用于集群数据同步的保存方法，保持原始时间戳
func SaveInstanceForSync(ins ServiceInstance) bool {
	mapLock.Lock()
	defer mapLock.Unlock()

	log.Printf("[storage] 集群同步保存服务实例: ServiceID=%s, 注册时间=%s, 心跳时间=%s, 注册时间戳=%d, 心跳时间戳=%d",
		ins.ServiceID, ins.RegisteredGMTTime, ins.LastHeartbeatGMTTime, ins.RegisteredAt, ins.LastHeartbeat)

	// 检查传入的实例是否有有效的时间戳
	if ins.RegisteredAt <= 0 || ins.LastHeartbeat <= 0 {
		log.Printf("[storage] 警告: 同步的实例 %s 时间戳无效 (注册时间=%d, 心跳时间=%d)，尝试保持现有时间戳",
			ins.ServiceID, ins.RegisteredAt, ins.LastHeartbeat)
	}

	// 检查ServiceID是否已存在
	for serviceName, list := range serviceMap {
		for i, exist := range list {
			if exist.ServiceID == ins.ServiceID {
				// 如果ServiceID已存在，智能合并信息
				log.Printf("[storage] 更新现有服务实例: ServiceID=%s, 原注册时间=%s, 新注册时间=%s",
					ins.ServiceID, exist.RegisteredGMTTime, ins.RegisteredGMTTime)

				// 智能合并：如果新数据的时间戳无效，保持原有时间戳
				updatedInstance := ins
				if ins.RegisteredAt <= 0 && exist.RegisteredAt > 0 {
					log.Printf("[storage] 保持原有注册时间戳: %d (%s)", exist.RegisteredAt, exist.RegisteredGMTTime)
					updatedInstance.RegisteredAt = exist.RegisteredAt
					updatedInstance.RegisteredGMTTime = exist.RegisteredGMTTime
				}
				if ins.LastHeartbeat <= 0 && exist.LastHeartbeat > 0 {
					log.Printf("[storage] 保持原有心跳时间戳: %d (%s)", exist.LastHeartbeat, exist.LastHeartbeatGMTTime)
					updatedInstance.LastHeartbeat = exist.LastHeartbeat
					updatedInstance.LastHeartbeatGMTTime = exist.LastHeartbeatGMTTime
				}

				serviceMap[serviceName][i] = updatedInstance
				return true
			}
		}
	}

	// ServiceID不存在，检查时间戳是否有效
	if ins.RegisteredAt <= 0 || ins.LastHeartbeat <= 0 {
		log.Printf("[storage] 警告: 新实例 %s 时间戳无效，设置为当前时间", ins.ServiceID)
		currentTime := time.Now().UTC()
		if ins.RegisteredAt <= 0 {
			ins.RegisteredAt = currentTime.Unix()
			ins.RegisteredGMTTime = currentTime.Format("2006-01-02 15:04:05")
		}
		if ins.LastHeartbeat <= 0 {
			ins.LastHeartbeat = currentTime.Unix()
			ins.LastHeartbeatGMTTime = currentTime.Format("2006-01-02 15:04:05")
		}
	}

	// ServiceID不存在，直接添加
	log.Printf("[storage] 添加新服务实例: ServiceID=%s, 注册时间=%s",
		ins.ServiceID, ins.RegisteredGMTTime)
	serviceMap[ins.ServiceName] = append(serviceMap[ins.ServiceName], ins)
	return true
}

// ServiceIDExists 检查ServiceID是否已存在
func ServiceIDExists(serviceID string) bool {
	mapLock.RLock()
	defer mapLock.RUnlock()

	for _, list := range serviceMap {
		for _, exist := range list {
			if exist.ServiceID == serviceID {
				return true
			}
		}
	}
	return false
}

func IPPortExists(ip string, port int) bool {
	mapLock.RLock()
	defer mapLock.RUnlock()

	for _, list := range serviceMap {
		for _, instance := range list {
			if instance.IPAddress == ip && instance.Port == port {
				return true
			}
		}
	}
	return false
}

// GetAllInstances 返回所有服务的所有实例
func GetAllInstances() []ServiceInstance {
	mapLock.RLock()
	defer mapLock.RUnlock()

	var result []ServiceInstance
	for _, list := range serviceMap {
		result = append(result, list...)
	}
	return result
}

// SelectOneInstance 使用轮询方式返回一个实例
func SelectOneInstance(serviceName string) *ServiceInstance {
	mapLock.Lock()
	defer mapLock.Unlock()

	list, ok := serviceMap[serviceName]
	if !ok || len(list) == 0 {
		return nil
	}

	index := rrIndexMap[serviceName]
	selected := list[index%len(list)]
	rrIndexMap[serviceName] = (index + 1) % len(list)

	return &selected
}

// GetInstanceByServiceID 根据ServiceID获取服务实例
func GetInstanceByServiceID(serviceID string) *ServiceInstance {
	mapLock.RLock()
	defer mapLock.RUnlock()

	for _, list := range serviceMap {
		for _, exist := range list {
			if exist.ServiceID == serviceID {
				return &exist
			}
		}
	}
	return nil
}

// RemoveInstance 删除服务实例
func RemoveInstance(serviceID string) bool {
	clusterMgr := getClusterManager()

	if clusterMgr != nil && clusterMgr.IsMaster() {
		// 主节点：先获取实例信息，删除后同步到从节点
		instance := GetInstanceByServiceID(serviceID)
		if instance == nil {
			return false
		}

		success := RemoveInstanceInternal(serviceID)
		if success {
			clusterMgr.SyncToSlaves("unregister", *instance)
		}
		return success
	} else if clusterMgr != nil {
		// 从节点：转发到主节点
		instance := GetInstanceByServiceID(serviceID)
		if instance == nil {
			return false
		}
		resp, err := clusterMgr.ForwardToMaster("unregister", *instance)
		if resp != nil {
			resp.Body.Close()
		}
		return err == nil
	} else {
		// 无集群管理，直接删除
		return RemoveInstanceInternal(serviceID)
	}
}

// RemoveInstanceInternal 内部删除方法，不触发同步
func RemoveInstanceInternal(serviceID string) bool {
	mapLock.Lock()
	defer mapLock.Unlock()

	for serviceName, list := range serviceMap {
		for i, exist := range list {
			if exist.ServiceID == serviceID {
				// 删除找到的实例
				serviceMap[serviceName] = append(list[:i], list[i+1:]...)
				// 如果该服务没有实例了，删除服务
				if len(serviceMap[serviceName]) == 0 {
					delete(serviceMap, serviceName)
					delete(rrIndexMap, serviceName)
				} else {
					// 重置轮询索引，防止索引越界
					rrIndexMap[serviceName] = 0
				}
				return true
			}
		}
	}
	return false
}

// UpdateHeartbeat 更新服务实例的心跳时间
func UpdateHeartbeat(serviceID string) bool {
	clusterMgr := getClusterManager()

	if clusterMgr != nil && clusterMgr.IsMaster() {
		// 主节点：直接更新并同步到从节点
		instance := GetInstanceByServiceID(serviceID)
		if instance == nil {
			return false
		}

		success := UpdateHeartbeatInternal(serviceID)
		if success {
			updatedInstance := GetInstanceByServiceID(serviceID)
			if updatedInstance != nil {
				clusterMgr.SyncToSlaves("heartbeat", *updatedInstance)
			}
		}
		return success
	} else if clusterMgr != nil {
		// 从节点：转发到主节点
		instance := GetInstanceByServiceID(serviceID)
		if instance == nil {
			return false
		}
		resp, err := clusterMgr.ForwardToMaster("heartbeat", *instance)
		if resp != nil {
			resp.Body.Close()
		}
		return err == nil
	} else {
		// 无集群管理，直接更新
		return UpdateHeartbeatInternal(serviceID)
	}
}

// UpdateHeartbeatInternal 内部心跳更新方法，不触发同步
func UpdateHeartbeatInternal(serviceID string) bool {
	mapLock.Lock()
	defer mapLock.Unlock()

	currentTime := time.Now().UTC()
	for serviceName, list := range serviceMap {
		for i, exist := range list {
			if exist.ServiceID == serviceID {
				// 更新心跳时间
				serviceMap[serviceName][i].LastHeartbeat = currentTime.Unix()
				serviceMap[serviceName][i].LastHeartbeatGMTTime = currentTime.Format("2006-01-02 15:04:05")
				return true
			}
		}
	}
	return false
}

// UpdateHeartbeatWithTime 使用指定的时间更新心跳（用于从主节点同步）
func UpdateHeartbeatWithTime(serviceID string, heartbeatTime int64, heartbeatGMTTime string) bool {
	mapLock.Lock()
	defer mapLock.Unlock()

	for serviceName, list := range serviceMap {
		for i, exist := range list {
			if exist.ServiceID == serviceID {
				// 使用传入的时间，保持与主节点一致
				serviceMap[serviceName][i].LastHeartbeat = heartbeatTime
				serviceMap[serviceName][i].LastHeartbeatGMTTime = heartbeatGMTTime
				return true
			}
		}
	}
	return false
}

// UpdateHeartbeatForResponse 更新心跳并返回主节点的响应（用于从节点心跳处理）
func UpdateHeartbeatForResponse(serviceID string, instance ServiceInstance) (*http.Response, bool) {
	clusterMgr := getClusterManager()

	if clusterMgr != nil && clusterMgr.IsMaster() {
		// 主节点：直接更新并同步到从节点
		success := UpdateHeartbeatInternal(serviceID)
		if success {
			updatedInstance := GetInstanceByServiceID(serviceID)
			if updatedInstance != nil {
				clusterMgr.SyncToSlaves("heartbeat", *updatedInstance)
			}
		}
		return nil, success
	} else if clusterMgr != nil {
		// 从节点：转发到主节点并返回响应
		resp, err := clusterMgr.ForwardToMaster("heartbeat", instance)
		return resp, err == nil
	} else {
		// 无集群管理，直接更新
		success := UpdateHeartbeatInternal(serviceID)
		return nil, success
	}
}

// UpdateInstanceHeartbeat 通过注册接口更新心跳时间（用于同步到peer）
func UpdateInstanceHeartbeat(ins ServiceInstance) bool {
	mapLock.Lock()
	defer mapLock.Unlock()

	for serviceName, list := range serviceMap {
		for i, exist := range list {
			if exist.ServiceID == ins.ServiceID {
				// 只更新心跳时间，不更新其他字段
				currentTime := time.Now()
				serviceMap[serviceName][i].LastHeartbeat = currentTime.UTC().Unix()
				serviceMap[serviceName][i].LastHeartbeatGMTTime = currentTime.UTC().Format("2006-01-02 15:04:05")
				return true
			}
		}
	}
	return false
}

// GetExpiredInstances 获取心跳超时的服务实例
func GetExpiredInstances() []ServiceInstance {
	mapLock.RLock()
	defer mapLock.RUnlock()

	var expiredInstances []ServiceInstance
	now := time.Now().UTC().Unix()
	timeoutSeconds := int64(config.GetHeartbeatTimeoutSeconds()) // 从配置文件读取超时时间

	log.Printf("[storage] 检查过期实例 - 当前UTC时间: %d, 超时阈值: %d秒", now, timeoutSeconds)

	for _, list := range serviceMap {
		for _, instance := range list {
			// 检查心跳时间戳是否有效
			if instance.LastHeartbeat <= 0 {
				log.Printf("[storage] 警告: 实例 %s 心跳时间戳无效 (%d)，跳过过期检查",
					instance.ServiceID, instance.LastHeartbeat)
				continue
			}

			timeDiff := now - instance.LastHeartbeat
			isExpired := timeDiff > timeoutSeconds

			log.Printf("[storage] 检查实例 %s: 最后心跳=%d(%s), 时间差=%d秒, 是否过期=%v",
				instance.ServiceID, instance.LastHeartbeat, instance.LastHeartbeatGMTTime, timeDiff, isExpired)

			if isExpired {
				expiredInstances = append(expiredInstances, instance)
			}
		}
	}

	log.Printf("[storage] 过期检测完成，发现 %d 个过期实例", len(expiredInstances))
	return expiredInstances
}

// StartExpiredInstanceCleanup 定时清理超时服务实例
func StartExpiredInstanceCleanup(interval time.Duration) {
	timeoutSeconds := config.GetHeartbeatTimeoutSeconds()
	log.Printf("[storage] 启动过期实例清理任务，检查间隔: %v，心跳超时: %d秒", interval, timeoutSeconds)
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			// 记录当前服务数量
			totalServices := GetTotalServiceCount()
			log.Printf("[storage] 当前总服务数量: %d", totalServices)

			expiredInstances := GetExpiredInstances()
			if len(expiredInstances) > 0 {
				log.Printf("[storage] 发现 %d 个过期服务实例", len(expiredInstances))
			}
			for _, instance := range expiredInstances {
				success := RemoveInstance(instance.ServiceID)
				if success {
					log.Printf("[success] 已注销超时服务实例: ServiceID=%s, ServiceName=%s, 最后心跳: %s",
						instance.ServiceID, instance.ServiceName, instance.LastHeartbeatGMTTime)
				} else {
					log.Printf("[error] 无法注销超时服务实例: ServiceID=%s, ServiceName=%s", instance.ServiceID, instance.ServiceName)
				}
			}
		}
	}()
}

// GetTotalServiceCount 获取当前总服务数量
func GetTotalServiceCount() int {
	mapLock.RLock()
	defer mapLock.RUnlock()

	count := 0
	for _, list := range serviceMap {
		count += len(list)
	}
	return count
}

// GetLoadBalanceState 获取所有服务的轮询状态
func GetLoadBalanceState() map[string]int {
	mapLock.RLock()
	defer mapLock.RUnlock()

	result := make(map[string]int)
	for serviceName, index := range rrIndexMap {
		result[serviceName] = index
	}
	return result
}

// SyncLoadBalanceStateFromMaster 从主节点同步轮询状态
func SyncLoadBalanceStateFromMaster(state map[string]int) {
	mapLock.Lock()
	defer mapLock.Unlock()

	log.Printf("[storage] 开始同步轮询状态，共 %d 个服务", len(state))
	syncCount := 0

	for serviceName, index := range state {
		// 检查服务是否存在，并且索引值合理
		if serviceList, exists := serviceMap[serviceName]; exists && len(serviceList) > 0 {
			// 确保索引值在合理范围内
			validIndex := index % len(serviceList)
			rrIndexMap[serviceName] = validIndex
			syncCount++
			log.Printf("[storage] 同步服务 %s 轮询索引: %d -> %d", serviceName, index, validIndex)
		}
	}

	log.Printf("[storage] 轮询状态同步完成，成功同步 %d 个服务的轮询状态", syncCount)
}

// ClusterManagerInterface 集群管理器接口，避免循环导入
type ClusterManagerInterface interface {
	IsMaster() bool
	SyncToSlaves(action string, instance ServiceInstance)
	ForwardToMaster(action string, instance ServiceInstance) (*http.Response, error)
}

var clusterManager ClusterManagerInterface

// SetClusterManager 设置集群管理器
func SetClusterManager(mgr ClusterManagerInterface) {
	clusterManager = mgr
}

// getClusterManager 获取集群管理器
func getClusterManager() ClusterManagerInterface {
	return clusterManager
}

// CheckAndCleanExpiredInstances 立即检查并清理过期实例（一次性操作）
func CheckAndCleanExpiredInstances() {
	log.Printf("[storage] 开始检查过期服务实例")

	totalServices := GetTotalServiceCount()
	log.Printf("[storage] 当前总服务数量: %d", totalServices)

	// 获取当前UTC时间用于调试
	currentUTC := time.Now().UTC()
	log.Printf("[storage] 当前UTC时间: %d (%s)", currentUTC.Unix(), currentUTC.Format("2006-01-02 15:04:05"))

	expiredInstances := GetExpiredInstances()
	if len(expiredInstances) > 0 {
		log.Printf("[storage] 发现 %d 个过期服务实例，开始清理", len(expiredInstances))
		for _, instance := range expiredInstances {
			log.Printf("[storage] 准备清理过期实例: ServiceID=%s, ServiceName=%s, 注册时间=%s, 最后心跳=%s",
				instance.ServiceID, instance.ServiceName, instance.RegisteredGMTTime, instance.LastHeartbeatGMTTime)

			success := RemoveInstance(instance.ServiceID)
			if success {
				log.Printf("[success] 已注销过期服务实例: ServiceID=%s, ServiceName=%s, 最后心跳: %s",
					instance.ServiceID, instance.ServiceName, instance.LastHeartbeatGMTTime)
			} else {
				log.Printf("[error] 无法注销过期服务实例: ServiceID=%s, ServiceName=%s", instance.ServiceID, instance.ServiceName)
			}
		}
	} else {
		log.Printf("[storage] 没有发现过期服务实例")
	}

	finalCount := GetTotalServiceCount()
	log.Printf("[storage] 清理后总服务数量: %d", finalCount)
}

// DebugPrintAllInstances 调试用：打印所有服务实例的详细信息
func DebugPrintAllInstances() {
	mapLock.RLock()
	defer mapLock.RUnlock()

	currentUTC := time.Now().UTC()
	log.Printf("[debug] ===== 当前所有服务实例详情 =====")
	log.Printf("[debug] 当前UTC时间: %d (%s)", currentUTC.Unix(), currentUTC.Format("2006-01-02 15:04:05"))

	count := 0
	for serviceName, instances := range serviceMap {
		log.Printf("[debug] 服务名: %s", serviceName)
		for i, instance := range instances {
			timeSinceHeartbeat := currentUTC.Unix() - instance.LastHeartbeat
			log.Printf("[debug]   实例[%d]: ServiceID=%s, IP=%s:%d",
				i, instance.ServiceID, instance.IPAddress, instance.Port)
			log.Printf("[debug]   注册时间: %d (%s)",
				instance.RegisteredAt, instance.RegisteredGMTTime)
			log.Printf("[debug]   心跳时间: %d (%s)",
				instance.LastHeartbeat, instance.LastHeartbeatGMTTime)
			log.Printf("[debug]   距离最后心跳: %d秒", timeSinceHeartbeat)
			count++
		}
	}

	if count == 0 {
		log.Printf("[debug] 没有任何服务实例")
	} else {
		log.Printf("[debug] 总计 %d 个服务实例", count)
	}
	log.Printf("[debug] ===== 服务实例详情结束 =====")
}
