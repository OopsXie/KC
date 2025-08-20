// Global variables
let currentPath = '/';
let fileList = [];
let healthCheckInterval = null;

// Initialize the application
document.addEventListener('DOMContentLoaded', function() {
    setupEventListeners();
    navigateTo('/');
    checkHealth();
    startHealthMonitoring();
});

// 页面卸载时清理定时器
window.addEventListener('beforeunload', function() {
    if (healthCheckInterval) {
        clearInterval(healthCheckInterval);
    }
});

// Setup event listeners
function setupEventListeners() {
    // File input
    document.getElementById('fileInput').addEventListener('change', handleFileUpload);
    
    // Upload area drag and drop
    const uploadArea = document.getElementById('uploadArea');
    uploadArea.addEventListener('click', () => document.getElementById('fileInput').click());
    uploadArea.addEventListener('dragover', handleDragOver);
    uploadArea.addEventListener('dragleave', handleDragLeave);
    uploadArea.addEventListener('drop', handleDrop);

    // Enter key for directory creation
    document.getElementById('dirName').addEventListener('keypress', function(e) {
        if (e.key === 'Enter') {
            createDirectory();
        }
    });
}

// API calls
async function apiCall(url, options = {}) {
    try {
        const response = await fetch(url, {
            ...options,
            headers: {
                'Content-Type': 'application/json',
                ...options.headers
            }
        });
        
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        
        return await response.json();
    } catch (error) {
        console.error('API call failed:', error);
        showError('网络请求失败: ' + error.message);
        throw error;
    }
}

// Navigate to directory
async function navigateTo(path) {
    currentPath = path;
    updateBreadcrumb(path);
    showLoading(true);
    
    try {
        const result = await apiCall(`/api/fs/list?path=${encodeURIComponent(path)}`);
        if (result.code === 200) {
            fileList = result.data || [];
            renderFileList();
            updateStatusBar();
        } else {
            showError(result.msg || '获取文件列表失败');
        }
    } catch (error) {
        showError('获取文件列表失败');
    } finally {
        showLoading(false);
    }
}

// Update breadcrumb
function updateBreadcrumb(path) {
    const breadcrumb = document.getElementById('breadcrumb');
    const parts = path.split('/').filter(part => part);
    
    let html = '<span class="breadcrumb-item" onclick="navigateTo(\'/\')"><i class="fas fa-home"></i> 根目录</span>';
    
    let currentPath = '';
    parts.forEach((part, index) => {
        currentPath += '/' + part;
        html += '<span class="breadcrumb-separator">/</span>';
        html += `<span class="breadcrumb-item" onclick="navigateTo('${currentPath}')">${part}</span>`;
    });
    
    breadcrumb.innerHTML = html;
}

// Render file list
function renderFileList() {
    const fileListContainer = document.getElementById('fileList');
    
    if (fileList.length === 0) {
        fileListContainer.innerHTML = '<div style="text-align: center; padding: 40px; color: #999;"><i class="fas fa-folder-open" style="font-size: 3rem; margin-bottom: 15px; display: block;"></i><p>此目录为空</p></div>';
        return;
    }
    
    let html = '';
    fileList.forEach(file => {
        const isDirectory = file.type === 'Directory';
        const icon = isDirectory ? 'fa-folder' : 'fa-file';
        const iconClass = isDirectory ? 'directory' : 'file';
        const fileName = file.path.split('/').pop();
        
        html += `
            <div class="file-item" ${isDirectory ? `onclick="navigateTo('${file.path}')"` : ''}>
                <div class="file-icon ${iconClass}">
                    <i class="fas ${icon}"></i>
                </div>
                <div class="file-info">
                    <div class="file-name">${fileName}</div>
                    <div class="file-meta">
                        ${file.formattedSize || formatFileSize(file.size)}
                    </div>
                </div>
                <div class="file-actions">
                    ${!isDirectory && file.type === 'File' ? `<button class="btn btn-info" onclick="event.stopPropagation(); showReplicaInfo('${file.path}')" title="查看副本分布"><i class="fas fa-copy"></i> 副本</button>` : ''}
                    ${!isDirectory ? `<button class="btn" onclick="event.stopPropagation(); downloadFile('${file.path}')"><i class="fas fa-download"></i></button>` : ''}
                    <button class="btn btn-danger" onclick="event.stopPropagation(); deleteFile('${file.path}')"><i class="fas fa-trash"></i></button>
                </div>
            </div>
        `;
    });
    
    fileListContainer.innerHTML = html;
}

// Create directory
async function createDirectory() {
    const dirName = document.getElementById('dirName').value.trim();
    if (!dirName) {
        showError('请输入目录名称');
        return;
    }
    
    const newPath = currentPath === '/' ? `/${dirName}` : `${currentPath}/${dirName}`;
    
    try {
        const result = await apiCall(`/api/fs/mkdir?path=${encodeURIComponent(newPath)}`, {
            method: 'POST'
        });
        
        if (result.code === 200) {
            showSuccess('目录创建成功');
            document.getElementById('dirName').value = '';
            navigateTo(currentPath);
        } else {
            showError(result.msg || '目录创建失败');
        }
    } catch (error) {
        showError('目录创建失败');
    }
}

// Handle file upload
async function handleFileUpload(event) {
    const files = event.target.files;
    if (files.length === 0) return;
    
    for (let file of files) {
        await uploadFile(file);
    }
    
    // Clear the input
    event.target.value = '';
    navigateTo(currentPath);
}

// Upload single file with progress
async function uploadFile(file) {
    const filePath = currentPath === '/' ? `/${file.name}` : `${currentPath}/${file.name}`;
    const formData = new FormData();
    formData.append('file', file);
    
    // Show progress modal
    showProgressModal('upload', file.name);
    
    try {
        const xhr = new XMLHttpRequest();
        currentXhr = xhr; // Store for cancellation
        
        // Track upload progress
        xhr.upload.onprogress = function(event) {
            if (event.lengthComputable) {
                const percentComplete = (event.loaded / event.total) * 100;
                updateProgress(percentComplete, event.loaded, event.total);
            }
        };
        
        // Handle completion
        xhr.onload = function() {
            if (xhr.status === 200) {
                try {
                    const result = JSON.parse(xhr.responseText);
                    if (result.code === 200) {
                        hideProgressModal();
                        showSuccess(`文件 ${file.name} 上传成功`);
                    } else {
                        hideProgressModal();
                        showError(`文件 ${file.name} 上传失败: ${result.msg}`);
                    }
                } catch (e) {
                    hideProgressModal();
                    showError(`文件 ${file.name} 上传失败`);
                }
            } else {
                hideProgressModal();
                showError(`文件 ${file.name} 上传失败`);
            }
        };
        
        // Handle errors
        xhr.onerror = function() {
            hideProgressModal();
            showError(`文件 ${file.name} 上传失败`);
        };
        
        // Start upload
        xhr.open('POST', `/api/fs/upload?path=${encodeURIComponent(filePath)}`);
        xhr.send(formData);
        
    } catch (error) {
        hideProgressModal();
        showError(`文件 ${file.name} 上传失败`);
    }
}

// Download file with progress
async function downloadFile(path) {
    const fileName = path.split('/').pop();
    
    // Show progress modal
    showProgressModal('download', fileName);
    
    try {
        const xhr = new XMLHttpRequest();
        currentXhr = xhr; // Store for cancellation
        xhr.responseType = 'blob';
        
        // Track download progress
        xhr.onprogress = function(event) {
            if (event.lengthComputable) {
                const percentComplete = (event.loaded / event.total) * 100;
                updateProgress(percentComplete, event.loaded, event.total);
            }
        };
        
        // Handle completion
        xhr.onload = function() {
            if (xhr.status === 200) {
                hideProgressModal();
                
                // Create download link
                const blob = xhr.response;
                const url = window.URL.createObjectURL(blob);
                const a = document.createElement('a');
                a.href = url;
                a.download = fileName;
                document.body.appendChild(a);
                a.click();
                window.URL.revokeObjectURL(url);
                document.body.removeChild(a);
                
                showSuccess('文件下载成功');
            } else {
                hideProgressModal();
                showError('文件下载失败');
            }
        };
        
        // Handle errors
        xhr.onerror = function() {
            hideProgressModal();
            showError('文件下载失败');
        };
        
        // Start download
        xhr.open('GET', `/api/fs/download?path=${encodeURIComponent(path)}`);
        xhr.send();
        
    } catch (error) {
        hideProgressModal();
        showError('文件下载失败');
    }
}

// Delete file
async function deleteFile(path) {
    const fileName = path.split('/').pop();
    if (!confirm(`确定要删除 "${fileName}" 吗？`)) {
        return;
    }
    
    try {
        const result = await apiCall(`/api/fs/delete?path=${encodeURIComponent(path)}`, {
            method: 'DELETE'
        });
        
        if (result.code === 200) {
            showSuccess('删除成功');
            navigateTo(currentPath);
        } else {
            showError(result.msg || '删除失败');
        }
    } catch (error) {
        showError('删除失败');
    }
}

// Refresh files
function refreshFiles() {
    navigateTo(currentPath);
}

// Check health
async function checkHealth() {
    try {
        const result = await apiCall('/api/fs/health');
        const statusElement = document.getElementById('connectionStatus');
        
        if (result.code === 200) {
            statusElement.textContent = '正常';
            statusElement.style.color = '#4caf50';
        } else {
            statusElement.textContent = '异常';
            statusElement.style.color = '#f44336';
        }
    } catch (error) {
        const statusElement = document.getElementById('connectionStatus');
        statusElement.textContent = '离线';
        statusElement.style.color = '#f44336';
    }
}

// Show cluster info - 跳转到专门的集群监控页面
function showClusterInfo() {
    window.open('cluster.html', '_blank');
}

// 开始健康状态监控
function startHealthMonitoring() {
    // 每60秒检查一次健康状态（比集群页面频率低一些）
    healthCheckInterval = setInterval(() => {
        checkHealth();
    }, 60000);
}

// Update status bar
function updateStatusBar() {
    const totalFiles = fileList.length;
    const totalSize = fileList.reduce((sum, file) => sum + (file.size || 0), 0);
    
    document.getElementById('totalFiles').textContent = totalFiles;
    document.getElementById('totalSize').textContent = formatFileSize(totalSize);
}

// Drag and drop handlers
function handleDragOver(e) {
    e.preventDefault();
    e.currentTarget.classList.add('dragover');
}

function handleDragLeave(e) {
    e.currentTarget.classList.remove('dragover');
}

function handleDrop(e) {
    e.preventDefault();
    e.currentTarget.classList.remove('dragover');
    
    const files = e.dataTransfer.files;
    if (files.length > 0) {
        handleFileUpload({ target: { files } });
    }
}

// Utility functions
function showLoading(show) {
    document.getElementById('loading').style.display = show ? 'block' : 'none';
    document.getElementById('fileList').style.display = show ? 'none' : 'block';
}

function showSuccess(message) {
    const alert = document.getElementById('alertSuccess');
    document.getElementById('successMessage').textContent = message;
    alert.style.display = 'block';
    setTimeout(() => alert.style.display = 'none', 3000);
}

function showError(message) {
    const alert = document.getElementById('alertError');
    document.getElementById('errorMessage').textContent = message;
    alert.style.display = 'block';
    setTimeout(() => alert.style.display = 'none', 5000);
}

function closeModal(modalId) {
    const modal = document.getElementById(modalId);
    if (modal) {
        modal.classList.remove('show');
    }
}

function formatFileSize(bytes) {
    if (!bytes || bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

function formatTime(timestamp) {
    if (!timestamp) return '未知时间';
    
    console.log('原始时间戳:', timestamp);
    
    // 尝试不同的时间戳格式
    let timestampMs;
    
    // 检查时间戳的数字大小来判断单位
    if (timestamp > 1e15) {
        // 如果时间戳大于10^15，可能是纳秒或其他超大单位，需要除以相应倍数
        // 纳秒转毫秒：除以 1,000,000
        timestampMs = timestamp / 1000000;
        console.log('检测为纳秒时间戳，转换后:', timestampMs);
    } else if (timestamp > 1e12) {
        // 如果时间戳大于10^12，可能是微秒，除以1000转换为毫秒
        timestampMs = timestamp / 1000;
        console.log('检测为微秒时间戳，转换后:', timestampMs);
    } else if (timestamp > 1e10) {
        // 如果时间戳大于10^10，可能已经是毫秒
        timestampMs = timestamp;
        console.log('检测为毫秒时间戳:', timestampMs);
    } else {
        // 标准Unix时间戳(秒)，转换为毫秒
        timestampMs = timestamp * 1000;
        console.log('检测为秒时间戳，转换后:', timestampMs);
    }
    
    const date = new Date(timestampMs);
    console.log('转换后的日期对象:', date);
    
    // 检查日期是否有效和合理（1970年到2100年之间）
    if (isNaN(date.getTime()) || date.getFullYear() < 1970 || date.getFullYear() > 2100) {
        console.error('时间戳转换结果异常:', {
            original: timestamp,
            converted: timestampMs,
            date: date,
            year: date.getFullYear()
        });
        return `时间错误(${timestamp})`;
    }
    
    // 明确指定上海时区，格式化为本地时间
    return date.toLocaleString('zh-CN', {
        timeZone: 'Asia/Shanghai',
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
        hour12: false  // 使用24小时制
    });
}

// Show replica distribution info
async function showReplicaInfo(filePath) {
    try {
        const result = await apiCall(`/api/fs/info?path=${encodeURIComponent(filePath)}`);
        if (result.code === 200 && result.data) {
            // 确保只对文件类型显示副本信息
            if (result.data.type === 'File') {
                displayReplicaModal(result.data);
            } else {
                showError('只有文件类型才有副本信息');
            }
        } else {
            showError('获取副本信息失败: ' + (result.msg || '未知错误'));
        }
    } catch (error) {
        showError('获取副本信息失败');
    }
}

// Display replica modal
function displayReplicaModal(fileInfo) {
    const modal = document.getElementById('replicaModal');
    const fileName = fileInfo.path.split('/').pop();
    
    document.getElementById('replicaModalTitle').textContent = `文件副本分布 - ${fileName}`;
    
    // 确保模态框有正确的CSS类
    const modalContent = modal.querySelector('.modal-content');
    if (modalContent && !modalContent.classList.contains('replica-modal')) {
        modalContent.classList.add('replica-modal');
    }
    
    let html = '';
    
    if (!fileInfo.replicaData || fileInfo.replicaData.length === 0) {
        html = '<div class="no-replicas"><i class="fas fa-info-circle"></i><p>该文件暂无副本信息</p></div>';
    } else {
        // 按DataServer节点分组
        const nodeGroups = {};
        fileInfo.replicaData.forEach(replica => {
            const node = replica.dsNode || 'Unknown';
            if (!nodeGroups[node]) {
                nodeGroups[node] = [];
            }
            nodeGroups[node].push(replica);
        });
        
        // Calculate redundancy level
        const redundancyLevel = fileInfo.replicaCount >= 3 ? 'high' : fileInfo.replicaCount >= 2 ? 'medium' : 'low';
        const redundancyText = redundancyLevel === 'high' ? '高冗余' : redundancyLevel === 'medium' ? '中等冗余' : '低冗余';
        const redundancyColor = redundancyLevel === 'high' ? 'var(--success-500)' : redundancyLevel === 'medium' ? 'var(--warning-500)' : 'var(--error-500)';
        
        html += `
            <div class="replica-summary">
                <div class="summary-item">
                    <i class="fas fa-copy"></i>
                    <div>
                        <div class="summary-value">${fileInfo.replicaCount}</div>
                        <div class="summary-label">总副本数</div>
                        <div class="summary-badge" style="background: ${redundancyColor}20; color: ${redundancyColor};">${redundancyText}</div>
                    </div>
                </div>
                <div class="summary-item">
                    <i class="fas fa-server"></i>
                    <div>
                        <div class="summary-value">${Object.keys(nodeGroups).length}</div>
                        <div class="summary-label">存储节点</div>
                        <div class="summary-note">分布式存储</div>
                    </div>
                </div>
                <div class="summary-item">
                    <i class="fas fa-hdd"></i>
                    <div>
                        <div class="summary-value">${fileInfo.formattedSize || formatFileSize(fileInfo.size)}</div>
                        <div class="summary-label">文件大小</div>
                        <div class="summary-note">× 3副本 = ${formatFileSize((fileInfo.size || 0) * 3)}</div>
                    </div>
                </div>
            </div>
        `;
        
        html += '<div class="replica-nodes">';
        
        Object.entries(nodeGroups).forEach(([node, replicas], index) => {
            const nodeClass = getNodeStatusClass(node);
            // 从节点地址中提取端口号
            const port = node.includes(':') ? node.split(':')[1] : (index + 1);
            html += `
                <div class="node-card ${nodeClass}">
                    <div class="node-header">
                        <div class="node-info">
                            <i class="fas fa-server"></i>
                            <div>
                                <div class="node-name">DataServer ${port}</div>
                                <div class="node-address">${node}</div>
                            </div>
                        </div>
                        <div class="node-status ${nodeClass}">
                            ${node !== 'Unknown' ? '在线' : '未知'}
                        </div>
                    </div>
                    <div class="node-replicas">
                        <div class="replica-count">${replicas.length} 个副本</div>
                        <div class="replica-list">
            `;
            
            replicas.forEach(replica => {
                html += `
                    <div class="replica-item">
                        <div class="replica-id">副本ID: ${replica.id || 'N/A'}</div>
                        <div class="replica-path">文件路径: ${fileInfo.path}</div>
                        <div class="replica-storage-path">存储路径: ${replica.path || 'N/A'}</div>
                    </div>
                `;
            });
            
            html += `
                        </div>
                    </div>
                </div>
            `;
        });
        
        html += '</div>';
    }
    
    document.getElementById('replicaContent').innerHTML = html;
    modal.classList.add('show');
    
    // 强制设置滚动属性（调试用）
    setTimeout(() => {
        const modalBody = modal.querySelector('.modal-body');
        if (modalBody) {
            modalBody.style.overflowY = 'auto';
            modalBody.style.maxHeight = 'calc(90vh - 120px)';
            console.log('副本模态框滚动属性已设置:', {
                overflowY: modalBody.style.overflowY,
                maxHeight: modalBody.style.maxHeight,
                scrollHeight: modalBody.scrollHeight,
                clientHeight: modalBody.clientHeight
            });
        }
    }, 100);
}

// Get node status class for styling
function getNodeStatusClass(node) {
    if (!node || node === 'Unknown') {
        return 'offline';
    }
    // 简单判断，实际可以通过集群信息API获取真实状态
    return 'online';
}

// Close modal when clicking outside
window.onclick = function(event) {
    const modals = document.getElementsByClassName('modal');
    for (let modal of modals) {
        if (event.target === modal) {
            modal.style.display = 'none';
        }
    }
}

// ===== Progress Bar Functions =====

let currentXhr = null;
let startTime = 0;

function showProgressModal(type, fileName) {
    const modal = document.getElementById('progressModal');
    const icon = document.getElementById('progressIcon');
    const iconSymbol = document.getElementById('progressIconSymbol');
    const title = document.getElementById('progressTitle');
    const fileNameElement = document.getElementById('progressFileName');
    
    // Check if all elements exist
    if (!modal || !icon || !iconSymbol || !title || !fileNameElement) {
        console.error('进度条模态框元素未找到，使用简单提示');
        showSuccess(type === 'upload' ? `开始上传 ${fileName}` : `开始下载 ${fileName}`);
        return;
    }
    
    // Set type-specific content
    if (type === 'upload') {
        icon.className = 'progress-icon upload';
        iconSymbol.className = 'fas fa-upload';
        title.textContent = '上传文件';
    } else if (type === 'download') {
        icon.className = 'progress-icon download';
        iconSymbol.className = 'fas fa-download';
        title.textContent = '下载文件';
    }
    
    fileNameElement.textContent = fileName;
    
    // Reset progress
    updateProgress(0, 0, 0);
    startTime = Date.now();
    
    // Show modal
    modal.classList.add('show');
}

function hideProgressModal() {
    const modal = document.getElementById('progressModal');
    if (modal) {
        modal.classList.remove('show');
    }
    currentXhr = null;
}

function updateProgress(percentage, loaded, total) {
    const progressBar = document.getElementById('progressBarFill');
    const progressPercentage = document.getElementById('progressPercentage');
    const progressSpeed = document.getElementById('progressSpeed');
    
    // Check if elements exist
    if (!progressBar || !progressPercentage || !progressSpeed) {
        return;
    }
    
    // Update progress bar
    progressBar.style.width = percentage + '%';
    progressPercentage.textContent = Math.round(percentage) + '%';
    
    // Calculate and display speed
    if (loaded > 0 && startTime > 0) {
        const elapsed = (Date.now() - startTime) / 1000; // seconds
        const speed = loaded / elapsed; // bytes per second
        progressSpeed.textContent = formatSpeed(speed);
    }
}

function formatSpeed(bytesPerSecond) {
    if (bytesPerSecond < 1024) {
        return bytesPerSecond.toFixed(0) + ' B/s';
    } else if (bytesPerSecond < 1024 * 1024) {
        return (bytesPerSecond / 1024).toFixed(1) + ' KB/s';
    } else {
        return (bytesPerSecond / (1024 * 1024)).toFixed(1) + ' MB/s';
    }
}

function cancelProgress() {
    if (currentXhr) {
        currentXhr.abort();
        currentXhr = null;
    }
    hideProgressModal();
    showError('操作已取消');
}
