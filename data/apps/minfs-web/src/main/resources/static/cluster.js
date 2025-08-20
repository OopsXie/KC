// 集群监控页面JavaScript

// 全局变量
let clusterData = null;
// 缓存最后已知的MetaServer角色信息（用于主从切换后的正确显示）
let lastKnownMetaRoles = new Map(); // port -> {role, isMaster}
let refreshInterval = null;

// 页面初始化
document.addEventListener('DOMContentLoaded', function() {
    loadClusterInfo();
    startAutoRefresh();
});

// 页面卸载时清理定时器
window.addEventListener('beforeunload', function() {
    if (refreshInterval) {
        clearInterval(refreshInterval);
    }
});

// 加载集群信息
async function loadClusterInfo() {
    showLoading(true);
    
    try {
        const response = await fetch('/api/fs/cluster');
        const result = await response.json();
        
        if (result.code === 200) {
            clusterData = result.data;
            console.log('🔍 完整集群数据:', clusterData);
            if (clusterData.dataServers) {
                console.log('🔍 DataServers数组:', clusterData.dataServers);
                clusterData.dataServers.forEach((ds, index) => {
                    console.log(`📊 DataServer ${index}:`, {
                        address: ds.address,
                        host: ds.host,
                        port: ds.port,
                        fileTotal: ds.fileTotal,
                        capacity: ds.capacity,
                        used: ds.used,
                        status: ds.status
                    });
                });
            }
            renderClusterInfo();
            updateLastUpdateTime();
            showSuccess('集群信息加载成功');
        } else {
            showError('获取集群信息失败: ' + result.msg);
        }
    } catch (error) {
        console.error('加载集群信息失败:', error);
        showError('网络连接失败，无法获取集群信息');
    } finally {
        showLoading(false);
    }
}

// 渲染集群信息
function renderClusterInfo() {
    if (!clusterData) return;
    
    // 更新概览卡片
    updateOverviewCards();
    
    // 渲染元数据服务器
    renderMetaServers();
    
    // 渲染数据服务器
    renderDataServers();
    
    // 更新集群状态
    updateClusterStatus();
}

// 更新概览卡片
function updateOverviewCards() {
    // 元数据服务器总数
    const totalMeta = clusterData.totalMetaServers || 0;
    document.getElementById('totalMetaServers').textContent = totalMeta;
    
    // 数据服务器总数
    const totalData = clusterData.totalDataServers || 0;
    document.getElementById('totalDataServers').textContent = totalData;
    
    // 计算存储使用率（模拟数据）
    const storageUsage = calculateStorageUsage();
    document.getElementById('storageUsage').textContent = storageUsage;
    
    // 更新集群状态
    const statusElement = document.getElementById('clusterStatus');
    if (totalMeta > 0 && totalData > 0) {
        statusElement.textContent = '正常运行';
        statusElement.style.color = 'var(--success-600)';
    } else {
        statusElement.textContent = '异常';
        statusElement.style.color = 'var(--error-600)';
    }
}

// 渲染元数据服务器
function renderMetaServers() {
    const container = document.getElementById('metaServerGrid');
    const badge = document.getElementById('metaServerBadge');
    
    // 预期的MetaServer配置（固定端口映射：1→9090, 2→9091, 3→9092）
    const expectedMetaServers = [
        { id: '1', port: 9090, host: 'localhost' },
        { id: '2', port: 9091, host: 'localhost' },
        { id: '3', port: 9092, host: 'localhost' }
    ];
    
    let runningCount = 0;
    let html = '';
    
    expectedMetaServers.forEach((expectedServer, index) => {
        let actualServer = null;
        let isRunning = false;
        let actualRole = 'Slave ' + (index + 1); // 默认角色，从1开始编号
        let isMaster = false;
        
        // 根据端口查找对应的运行中服务器
        if (clusterData) {
            // 检查是否为Master（根据实际端口匹配）
            if (clusterData.masterMetaServer && clusterData.masterMetaServer.port === expectedServer.port) {
                actualServer = clusterData.masterMetaServer;
                actualRole = 'Master';
                isMaster = true;
                isRunning = true;
                runningCount++;
                // 缓存当前角色信息
                lastKnownMetaRoles.set(expectedServer.port, { role: 'Master', isMaster: true });
            } 
            // 检查是否为Slave（根据实际端口匹配）
            else if (clusterData.slaveMetaServers) {
                const slaveServer = clusterData.slaveMetaServers.find(slave => slave.port === expectedServer.port);
                if (slaveServer) {
                    actualServer = slaveServer;
                    // 根据在slave列表中的位置确定角色编号
                    const slaveIndex = clusterData.slaveMetaServers.findIndex(slave => slave.port === expectedServer.port);
                    actualRole = 'Slave ' + (slaveIndex + 1);
                    isMaster = false;
                    isRunning = true;
                    runningCount++;
                    // 缓存当前角色信息
                    lastKnownMetaRoles.set(expectedServer.port, { role: actualRole, isMaster: false });
                }
            }
        }
        
        // 如果没有运行，尝试使用缓存的角色信息，否则使用默认角色
        if (!isRunning) {
            const cachedRole = lastKnownMetaRoles.get(expectedServer.port);
            if (cachedRole) {
                actualRole = cachedRole.role;
                isMaster = cachedRole.isMaster;
            } else {
                // 默认角色（基于端口顺序）
                if (index === 0) {
                    actualRole = 'Master';
                    isMaster = true;
                } else {
                    actualRole = 'Slave ' + index;
                    isMaster = false;
                }
            }
        }
        
        // 创建服务器信息对象 - 始终显示预期的端口
        const serverInfo = {
            address: `${expectedServer.host}:${expectedServer.port}`,
            status: isRunning ? 'Active' : 'Stopped',
            host: expectedServer.host,
            port: expectedServer.port
        };
        
        // 检查是否使用了缓存的角色信息
        const usingCachedRole = !isRunning && lastKnownMetaRoles.has(expectedServer.port);
        
        // 调试信息：记录端口和ID的映射关系
        console.log(`MetaServer 映射 - 端口: ${expectedServer.port}, ID: ${expectedServer.id}, 角色: ${actualRole}, 运行状态: ${isRunning}`);
        
        html += createServerCard(
            serverInfo,
            actualRole,
            'meta-server',
            isMaster,
            expectedServer.id,
            isRunning,
            usingCachedRole
        );
    });
    
    badge.textContent = `${runningCount}/${expectedMetaServers.length} 台在线`;
    container.innerHTML = html;
}

// 渲染数据服务器
function renderDataServers() {
    const container = document.getElementById('dataServerGrid');
    const badge = document.getElementById('dataServerBadge');
    
    // 预期的DataServer配置（固定端口映射：1→8001, 2→8002, 3→8003, 4→8004）
    const expectedDataServers = [
        { id: '1', port: 8001, host: 'localhost' },
        { id: '2', port: 8002, host: 'localhost' },
        { id: '3', port: 8003, host: 'localhost' },
        { id: '4', port: 8004, host: 'localhost' }
    ];
    
    let runningCount = 0;
    let html = '';
    
    expectedDataServers.forEach((expectedServer, index) => {
        let actualServer = null;
        let isRunning = false;
        
        // 查找对应的运行中服务器
        if (clusterData && clusterData.dataServers) {
            actualServer = clusterData.dataServers.find(server => {
                const serverPort = server.address ? parseInt(server.address.split(':')[1]) : 0;
                return serverPort === expectedServer.port;
            });
            
            if (actualServer) {
                isRunning = true;
                runningCount++;
            }
        }
        
        // 创建服务器信息对象 - 始终显示预期的端口，但使用实际的容量数据
        const serverInfo = {
            address: `${expectedServer.host}:${expectedServer.port}`,
            status: isRunning ? 'Active' : 'Stopped',
            capacity: actualServer ? actualServer.capacity : 0,
            used: actualServer ? actualServer.used : 0,
            host: expectedServer.host,
            port: expectedServer.port
        };
        
        console.log(`DataServer 映射 - 端口: ${expectedServer.port}, ID: ${expectedServer.id}, 运行状态: ${isRunning}`);
        console.log(`🔍 ActualServer数据:`, actualServer);
        if (actualServer) {
            console.log(`📊 ActualServer详细信息:`, {
                host: actualServer.host,
                port: actualServer.port,
                address: actualServer.address,
                fileTotal: actualServer.fileTotal,
                capacity: actualServer.capacity,
                used: actualServer.used
            });
        }
        
        html += createDataServerCard(serverInfo, index + 1, expectedServer.id, isRunning, actualServer);
    });
    
    badge.textContent = `${runningCount}/${expectedDataServers.length} 台在线`;
    container.innerHTML = html;
}

// 创建服务器卡片
function createServerCard(server, role, type, isMaster, serverId, isRunning, usingCachedRole = false) {
    const isActive = isRunning !== undefined ? isRunning : (server.status === 'Active');
    const statusClass = isActive ? 'active' : 'inactive';
    const statusText = isActive ? '在线' : '已停止';
    const statusBadgeClass = isActive ? 'active' : 'stopped';
    const serverType = type === 'meta-server' ? 'meta' : 'data';
    // 确保使用正确的服务器ID，优先使用传入的serverId
    const actualServerId = serverId || '1';
    
    // 为MetaServer添加角色提示
    const roleHint = type === 'meta-server' && usingCachedRole ? 
        '<small class="role-hint" title="显示的是停止前的角色，主从关系可能已发生变化">⚠️ 角色可能已变更</small>' : '';
    
    return `
        <div class="server-card ${statusClass}">
            <div class="server-header">
                <div class="server-title">
                    <i class="fas fa-${type === 'meta-server' ? 'database' : 'hdd'}"></i>
                    ${role}${isMaster ? ' (主)' : ''}
                    ${roleHint}
                </div>
                <div class="server-status ${statusBadgeClass}">${statusText}</div>
            </div>
            <div class="server-info">
                <div class="info-row">
                    <span class="info-label">地址</span>
                    <span class="info-value">${server.address || 'N/A'}</span>
                </div>
                <div class="info-row">
                    <span class="info-label">状态</span>
                    <span class="info-value">${server.status || 'Unknown'}</span>
                </div>
                <div class="info-row">
                    <span class="info-label">类型</span>
                    <span class="info-value">${isMaster ? '主服务器' : '从服务器'}</span>
                </div>
                ${type === 'meta-server' && usingCachedRole ? 
                    '<div class="info-row"><span class="info-label">提示</span><span class="info-value text-warning">停止前角色，可能已变更</span></div>' : ''}
            </div>
            <div class="server-actions">
                <button class="btn-sm btn-success" onclick="controlSingleServer('${serverType}', 'start', '${actualServerId}')" 
                        title="启动服务器" ${isActive ? 'disabled' : ''}>
                    <i class="fas fa-play"></i>
                </button>
                <button class="btn-sm btn-danger" onclick="controlSingleServer('${serverType}', 'stop', '${actualServerId}')" 
                        title="停止服务器" ${!isActive ? 'disabled' : ''}>
                    <i class="fas fa-stop"></i>
                </button>

                <button class="btn-sm" onclick="controlSingleServer('${serverType}', 'status', '${actualServerId}')" 
                        title="查看状态">
                    <i class="fas fa-info-circle"></i>
                </button>
            </div>
        </div>
    `;
}

// 创建数据服务器卡片
function createDataServerCard(server, index, serverId, isRunning, actualServerData = null) {
    const isActive = isRunning !== undefined ? isRunning : (server.status === 'Active');
    const statusClass = isActive ? 'active' : 'inactive';
    const statusText = isActive ? '在线' : '已停止';
    const statusBadgeClass = isActive ? 'active' : 'stopped';
    const actualServerId = serverId || index.toString();
    
    // 使用真实的容量数据
    const capacity = server.capacity || 0;
    const used = server.used || 0;
    
    // 计算使用率
    let usagePercentage = 0;
    if (capacity > 0) {
        usagePercentage = ((used / capacity) * 100).toFixed(1);
    }
    const usageClass = usagePercentage > 80 ? 'high' : '';
    
    return `
        <div class="server-card ${statusClass}">
            <div class="server-header">
                <div class="server-title">
                    <i class="fas fa-hdd"></i>
                    数据服务器 ${index}
                </div>
                <div class="server-status ${statusBadgeClass}">${statusText}</div>
            </div>
            <div class="server-info">
                <div class="info-row">
                    <span class="info-label">地址</span>
                    <span class="info-value">${server.address || `DataServer-${index}`}</span>
                </div>
                <div class="info-row">
                    <span class="info-label">状态</span>
                    <span class="info-value">${server.status || 'Active'}</span>
                </div>
                <div class="info-row">
                    <span class="info-label">容量</span>
                    <span class="info-value">${capacity > 0 ? formatBytes(capacity * 1024 * 1024) : (isActive ? '未知' : '离线')}</span>
                </div>
                <div class="info-row">
                    <span class="info-label">已使用</span>
                    <span class="info-value">${capacity > 0 ? `${formatBytes(used * 1024 * 1024)} (${usagePercentage}%)` : (isActive ? '未知' : '离线')}</span>
                </div>
                <div class="info-row">
                    <span class="info-label">文件块数</span>
                    <span class="info-value">${(() => {
                        console.log(`🔍 文件块数检查 - actualServerData:`, actualServerData);
                        console.log(`🔍 fileTotal值:`, actualServerData ? actualServerData.fileTotal : 'N/A');
                        console.log(`🔍 isActive:`, isActive);
                        return actualServerData && actualServerData.fileTotal ? actualServerData.fileTotal : (isActive ? '未知' : '离线');
                    })()}</span>
                </div>

            </div>
            <div class="server-actions">
                <button class="btn-sm btn-success" onclick="controlSingleServer('data', 'start', '${actualServerId}')" 
                        title="启动服务器" ${isActive ? 'disabled' : ''}>
                    <i class="fas fa-play"></i>
                </button>
                <button class="btn-sm btn-danger" onclick="controlSingleServer('data', 'stop', '${actualServerId}')" 
                        title="停止服务器" ${!isActive ? 'disabled' : ''}>
                    <i class="fas fa-stop"></i>
                </button>

                <button class="btn-sm" onclick="controlSingleServer('data', 'status', '${actualServerId}')" 
                        title="查看状态">
                    <i class="fas fa-info-circle"></i>
                </button>
                <button class="btn-sm btn-primary" onclick="showDataServerDetails('${server.address}', '${server.port}')" 
                        title="查看详情" ${!isActive ? 'disabled' : ''}>
                    <i class="fas fa-eye"></i>
                </button>
            </div>
        </div>
    `;
}

// 计算总存储使用率（使用真实数据）
function calculateStorageUsage() {
    if (!clusterData.dataServers || clusterData.dataServers.length === 0) {
        return '0%';
    }
    
    let totalCapacity = 0;
    let totalUsed = 0;
    let validServers = 0;
    
    // 统计所有数据服务器的容量和使用量
    clusterData.dataServers.forEach(server => {
        if (server.capacity > 0) {
            totalCapacity += server.capacity;
            totalUsed += server.used || 0;
            validServers++;
        }
    });
    
    if (totalCapacity === 0 || validServers === 0) {
        return '未知';
    }
    
    const usagePercentage = ((totalUsed / totalCapacity) * 100).toFixed(1);
    return `${usagePercentage}%`;
}

// 更新集群状态
function updateClusterStatus() {
    const metaCount = clusterData.totalMetaServers || 0;
    const dataCount = clusterData.totalDataServers || 0;
    
    // 检查健康状态
    if (metaCount > 0 && dataCount > 0) {
        updateStatusIndicator('healthy');
    } else {
        updateStatusIndicator('unhealthy');
    }
}

// 更新状态指示器
function updateStatusIndicator(status) {
    const elements = document.querySelectorAll('.cluster-status');
    elements.forEach(el => {
        el.className = `cluster-status ${status}`;
    });
}

// 刷新集群信息
async function refreshClusterInfo() {
    await loadClusterInfo();
}

// 开始自动刷新
function startAutoRefresh() {
    // 每30秒刷新一次
    refreshInterval = setInterval(() => {
        loadClusterInfo();
    }, 30000);
}

// 更新最后更新时间
function updateLastUpdateTime() {
    const now = new Date();
    const timeString = now.toLocaleString('zh-CN');
    document.getElementById('lastUpdate').textContent = timeString;
}

// 显示加载状态
function showLoading(show) {
    const overlay = document.getElementById('loadingOverlay');
    if (show) {
        overlay.classList.add('show');
    } else {
        overlay.classList.remove('show');
    }
}

// 显示成功消息
function showSuccess(message) {
    const alert = document.getElementById('alertSuccess');
    document.getElementById('successMessage').textContent = message;
    alert.style.display = 'block';
    setTimeout(() => alert.style.display = 'none', 3000);
}

// 显示错误消息
function showError(message) {
    const alert = document.getElementById('alertError');
    document.getElementById('errorMessage').textContent = message;
    alert.style.display = 'block';
    setTimeout(() => alert.style.display = 'none', 5000);
}

// 格式化字节数
function formatBytes(bytes) {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

// 为空状态添加样式
const emptyStateStyle = `
    .empty-state {
        text-align: center;
        padding: var(--spacing-10);
        color: var(--gray-500);
        grid-column: 1 / -1;
    }
    
    .empty-state i {
        font-size: 3rem;
        margin-bottom: var(--spacing-4);
        display: block;
        color: var(--gray-400);
    }
    
    .empty-state p {
        font-size: 1.125rem;
        font-weight: 600;
        margin: 0;
    }
`;

// 添加样式到页面
const style = document.createElement('style');
style.textContent = emptyStateStyle;
document.head.appendChild(style);

// ==================== 服务器控制功能 ====================

// 显示批量控制模态框
function showBatchControlModal() {
    const modal = document.getElementById('serverControlModal');
    if (modal) {
        modal.classList.add('show');
    }
}

// 关闭模态框
function closeModal(modalId) {
    const modal = document.getElementById(modalId);
    if (modal) {
        modal.classList.remove('show');
    }
}

// 批量控制服务器
async function controlServer(serverType, action) {
    const serverIdInput = serverType === 'meta' ? 'metaServerId' : 'dataServerId';
    const serverId = document.getElementById(serverIdInput).value.trim();
    
    if (serverId) {
        // 控制单个指定服务器
        await executeServerControl(serverType, action, serverId);
    } else {
        // 批量控制所有服务器
        await batchControlServers(serverType, action);
    }
}

// 批量控制所有服务器
async function batchControlServers(serverType, action) {
    const actionText = {
        'start': '启动',
        'stop': '停止', 
        'status': '查看状态'
    };
    
    const serverText = serverType === 'meta' ? 'MetaServer' : 'DataServer';
    
    // 确定要控制的服务器列表
    let serverIds = [];
    if (serverType === 'meta') {
        serverIds = ['1', '2', '3']; // MetaServer 1-3
    } else {
        serverIds = ['1', '2', '3', '4']; // DataServer 1-4
    }
    
    // 显示批量操作提示
    showLoading(true);
    showSuccess(`开始批量${actionText[action]}所有${serverText}...`);
    
    let successCount = 0;
    let failCount = 0;
    const results = [];
    
    // 根据操作类型决定是否并行执行
    if (action === 'stop') {
        // 停止操作：并行执行（可以同时停止多个服务）
        const promises = serverIds.map(async (id) => {
            try {
                await executeServerControl(serverType, action, id);
                successCount++;
                results.push(`${serverText} ${id}: 成功`);
            } catch (error) {
                failCount++;
                results.push(`${serverText} ${id}: 失败 - ${error.message}`);
            }
        });
        
        await Promise.allSettled(promises);
    } else {
        // 启动操作：串行执行（避免资源竞争）
        for (const id of serverIds) {
            try {
                await executeServerControl(serverType, action, id);
                successCount++;
                results.push(`${serverText} ${id}: 成功`);
                
                // 启动操作之间添加延迟，避免端口冲突
                if (action === 'start') {
                    await new Promise(resolve => setTimeout(resolve, 2000));
                }
            } catch (error) {
                failCount++;
                results.push(`${serverText} ${id}: 失败 - ${error.message}`);
                
                // 如果启动失败，询问是否继续
                if (action === 'start' && !confirm(`${serverText} ${id} 启动失败，是否继续启动其他服务器？\n错误：${error.message}`)) {
                    break;
                }
            }
        }
    }
    
    showLoading(false);
    
    // 显示批量操作结果
    const summaryMessage = `批量${actionText[action]}完成：成功 ${successCount} 个，失败 ${failCount} 个`;
    
    if (failCount === 0) {
        showSuccess(summaryMessage);
    } else {
        showError(summaryMessage);
    }
    
    // 显示详细结果
    showBatchOperationResults(results, `批量${actionText[action]}${serverText}结果`);
    
    // 刷新集群信息
    setTimeout(() => {
        loadClusterInfo();
    }, 3000);
}

// 单个服务器控制
async function controlSingleServer(serverType, action, serverId) {
    console.log(`🔧 控制服务器 - 类型: ${serverType}, 操作: ${action}, 服务器ID: ${serverId}`);
    
    // 将逻辑ID转换为实际端口号（后端脚本需要端口号）
    let actualServerId = serverId;
    
    if (serverType === 'meta') {
        const expectedMetaServers = [
            { id: '1', port: 9090, host: 'localhost' },
            { id: '2', port: 9091, host: 'localhost' },
            { id: '3', port: 9092, host: 'localhost' }
        ];
        const targetServer = expectedMetaServers.find(s => s.id === serverId);
        if (targetServer) {
            actualServerId = targetServer.port.toString(); // 转换为端口号字符串
            console.log(`🎯 目标MetaServer - ID: ${serverId} → 端口: ${targetServer.port}`);
        }
    } else if (serverType === 'data') {
        const expectedDataServers = [
            { id: '1', port: 8001, host: 'localhost' },
            { id: '2', port: 8002, host: 'localhost' },
            { id: '3', port: 8003, host: 'localhost' },
            { id: '4', port: 8004, host: 'localhost' }
        ];
        const targetServer = expectedDataServers.find(s => s.id === serverId);
        if (targetServer) {
            actualServerId = targetServer.port.toString(); // 转换为端口号字符串
            console.log(`🎯 目标DataServer - ID: ${serverId} → 端口: ${targetServer.port}`);
        }
    }
    
    console.log(`📡 发送给后端的serverId: ${actualServerId}`);
    await executeServerControl(serverType, action, actualServerId);
}

// 防重复操作的状态管理
const operationStates = new Map();

// 执行服务器控制命令
async function executeServerControl(serverType, action, serverId) {
    const actionText = {
        'start': '启动',
        'stop': '停止', 
        'status': '查看状态'
    };
    
    const serverText = serverType === 'meta' ? 'MetaServer' : 'DataServer';
    const serverIdText = serverId ? ` ${serverId}` : '';
    
    // 创建操作唯一标识
    const operationKey = `${serverType}_${action}_${serverId}`;
    
    // 检查是否已有相同操作在进行
    if (operationStates.has(operationKey)) {
        showError(`${actionText[action]} ${serverText}${serverIdText} 操作正在进行中，请稍等...`);
        return;
    }
    
    // 标记操作开始
    operationStates.set(operationKey, true);
    
    // 禁用相关按钮
    disableServerButtons(serverType, serverId, true);
    
    showLoading(true);
    
    try {
        let url;
        const params = new URLSearchParams();
        if (serverId) {
            params.append('serverId', serverId);
        }
        
        if (action === 'status') {
            url = `/api/fs/server/status?serverType=${serverType}`;
        } else {
            url = `/api/fs/server/${serverType}/${action}`;
        }
        
        if (serverId) {
            url += url.includes('?') ? '&' : '?';
            url += `serverId=${serverId}`;
        }
        
        const response = await fetch(url, {
            method: action === 'status' ? 'GET' : 'POST',
            headers: {
                'Content-Type': 'application/json'
            }
        });
        
        const result = await response.json();
        
        if (result.code === 200) {
            showSuccess(`${actionText[action]} ${serverText}${serverIdText} 成功`);
            
            // 显示命令输出（特别是状态查询）
            if (result.data && result.data.trim()) {
                showCommandOutput(result.data);
            }
            
            // 如果是启动/停止操作，刷新集群信息
            if (['start', 'stop'].includes(action)) {
                setTimeout(() => {
                    loadClusterInfo();
                }, 2000); // 延迟2秒刷新
            }
        } else {
            showError(`${actionText[action]} ${serverText}${serverIdText} 失败: ${result.msg}`);
        }
    } catch (error) {
        console.error('服务器控制失败:', error);
        showError(`${actionText[action]} ${serverText}${serverIdText} 失败: 网络错误`);
    } finally {
        showLoading(false);
        closeModal('serverControlModal');
        
        // 清除操作状态
        operationStates.delete(operationKey);
        
        // 重新启用按钮
        disableServerButtons(serverType, serverId, false);
    }
}

// 禁用/启用服务器控制按钮
function disableServerButtons(serverType, serverId, disable) {
    // 查找所有相关的按钮并禁用/启用
    const buttons = document.querySelectorAll(`[onclick*="controlSingleServer('${serverType}',"][onclick*="'${serverId}')"]`);
    buttons.forEach(button => {
        if (disable) {
            button.disabled = true;
            button.style.opacity = '0.5';
            button.style.cursor = 'not-allowed';
        } else {
            button.disabled = false;
            button.style.opacity = '';
            button.style.cursor = '';
        }
    });
}

// 显示命令输出
function showCommandOutput(output) {
    document.getElementById('commandOutput').textContent = output;
    const modal = document.getElementById('commandOutputModal');
    if (modal) {
        modal.classList.add('show');
    }
}

// 显示批量操作结果
function showBatchOperationResults(results, title) {
    const modal = document.getElementById('commandOutputModal');
    const titleElement = modal.querySelector('.modal-header h3');
    const outputElement = document.getElementById('commandOutput');
    
    titleElement.textContent = title;
    outputElement.textContent = results.join('\n');
    if (modal) {
        modal.classList.add('show');
    }
}

// 模态框点击外部关闭
window.onclick = function(event) {
    const modals = document.getElementsByClassName('modal');
    for (let modal of modals) {
        if (event.target === modal) {
            modal.classList.remove('show');
        }
    }
}

// 键盘快捷键支持
document.addEventListener('keydown', function(event) {
    // ESC键关闭模态框
    if (event.key === 'Escape') {
        const modals = document.getElementsByClassName('modal');
        for (let modal of modals) {
            if (modal.classList.contains('show')) {
                modal.classList.remove('show');
            }
        }
    }
});

// ==================== DataServer 详情功能 ====================

// 显示DataServer详情
function showDataServerDetails(address, port) {
    console.log('🔍 显示DataServer详情 - 地址:', address, '端口:', port);
    
    // 从全局clusterData中查找对应的DataServer数据
    let serverData = null;
    
    if (clusterData && clusterData.dataServers) {
        serverData = clusterData.dataServers.find(server => {
            const serverPort = server.address ? parseInt(server.address.split(':')[1]) : server.port;
            return serverPort === parseInt(port);
        });
        
        console.log('🔍 从集群数据中找到的服务器:', serverData);
    }
    
    if (!serverData) {
        console.error('❌ 无法在集群数据中找到对应的DataServer');
        showError('无法获取服务器详细信息，可能服务器离线或数据不可用');
        return;
    }
    
    console.log('✅ 使用的服务器数据:', {
        address: serverData.address,
        host: serverData.host,
        port: serverData.port,
        fileTotal: serverData.fileTotal,
        capacity: serverData.capacity,
        used: serverData.used,
        status: serverData.status
    });
    
    // 设置模态框标题
    const titleElement = document.getElementById('dataServerDetailsTitle');
    if (titleElement) {
        titleElement.textContent = `DataServer 详情 - ${address}`;
    }
    
    // 生成详情内容
    const detailsHtml = generateDataServerDetailsHtml(serverData);
    const contentElement = document.getElementById('dataServerDetailsContent');
    if (contentElement) {
        contentElement.innerHTML = detailsHtml;
    }
    
    // 显示模态框
    const modal = document.getElementById('dataServerDetailsModal');
    if (modal) {
        modal.classList.add('show');
    } else {
        console.error('找不到DataServer详情模态框元素');
        showError('模态框显示失败');
    }
}

// 生成DataServer详情HTML
function generateDataServerDetailsHtml(serverData) {
    const capacity = serverData.capacity || 0;
    const used = serverData.used || 0;  // 修正字段名
    const usagePercentage = capacity > 0 ? ((used / capacity) * 100).toFixed(1) : 0;
    const freeSpace = capacity - used;
    
    return `
        <div class="dataserver-details">
            <!-- 基本信息 -->
            <div class="details-section">
                <h4><i class="fas fa-server"></i> 基本信息</h4>
                <div class="details-grid">
                    <div class="detail-item">
                        <div class="detail-label">服务器地址</div>
                        <div class="detail-value">${serverData.host}:${serverData.port}</div>
                    </div>
                    <div class="detail-item">
                        <div class="detail-label">主机</div>
                        <div class="detail-value">${serverData.host}</div>
                    </div>
                    <div class="detail-item">
                        <div class="detail-label">端口</div>
                        <div class="detail-value">${serverData.port}</div>
                    </div>
                    <div class="detail-item">
                        <div class="detail-label">状态</div>
                        <div class="detail-value status-active">运行中</div>
                    </div>
                </div>
            </div>
            
            <!-- 存储信息 -->
            <div class="details-section">
                <h4><i class="fas fa-hdd"></i> 存储信息</h4>
                <div class="storage-overview">
                    <div class="storage-chart">
                        <div class="storage-pie">
                            <div class="pie-segment used" style="--percentage: ${usagePercentage}%"></div>
                        </div>
                        <div class="storage-center">
                            <div class="usage-text">${usagePercentage}%</div>
                            <div class="usage-label">已使用</div>
                        </div>
                    </div>
                    <div class="storage-stats">
                        <div class="storage-stat">
                            <div class="stat-icon total">
                                <i class="fas fa-database"></i>
                            </div>
                            <div class="stat-info">
                                <div class="stat-value">${formatBytes(capacity)}</div>
                                <div class="stat-label">总容量</div>
                            </div>
                        </div>
                        <div class="storage-stat">
                            <div class="stat-icon used">
                                <i class="fas fa-chart-pie"></i>
                            </div>
                            <div class="stat-info">
                                <div class="stat-value">${formatBytes(used * 1024 * 1024)}</div>
                                <div class="stat-label">已使用</div>
                            </div>
                        </div>
                        <div class="storage-stat">
                            <div class="stat-icon free">
                                <i class="fas fa-archive"></i>
                            </div>
                            <div class="stat-info">
                                <div class="stat-value">${formatBytes(freeSpace * 1024 * 1024)}</div>
                                <div class="stat-label">可用空间</div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
            
            <!-- 文件信息 -->
            <div class="details-section">
                <h4><i class="fas fa-file"></i> 文件信息</h4>
                <div class="file-stats">
                    <div class="file-stat-card">
                        <div class="file-stat-icon">
                            <i class="fas fa-cubes"></i>
                        </div>
                        <div class="file-stat-content">
                            <div class="file-stat-value">${serverData.fileTotal || 0}</div>
                            <div class="file-stat-label">文件块总数</div>
                        </div>
                    </div>
                    <div class="file-stat-card">
                        <div class="file-stat-icon">
                            <i class="fas fa-copy"></i>
                        </div>
                        <div class="file-stat-content">
                            <div class="file-stat-value">-</div>
                            <div class="file-stat-label">副本数量</div>
                            <div class="file-stat-note">需要分析文件副本</div>
                        </div>
                    </div>
                </div>
            </div>
            
            <!-- 性能指标 -->
            <div class="details-section">
                <h4><i class="fas fa-tachometer-alt"></i> 性能指标</h4>
                <div class="performance-grid">
                    <div class="performance-item">
                        <div class="performance-label">存储使用率</div>
                        <div class="performance-bar">
                            <div class="bar-fill" style="width: ${usagePercentage}%; background-color: ${usagePercentage > 80 ? '#ef4444' : usagePercentage > 60 ? '#f59e0b' : '#22c55e'}"></div>
                        </div>
                        <div class="performance-value">${usagePercentage}%</div>
                    </div>
                    <div class="performance-item">
                        <div class="performance-label">平均文件大小</div>
                        <div class="performance-value">${serverData.fileTotal > 0 ? formatBytes((used * 1024 * 1024) / serverData.fileTotal) : '0 B'}</div>
                    </div>
                </div>
            </div>
        </div>
    `;
}
