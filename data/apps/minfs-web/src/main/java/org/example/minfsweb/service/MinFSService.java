package org.example.minfsweb.service;


import com.ksyun.campus.client.EFileSystem;
import com.ksyun.campus.client.FSInputStream;
import com.ksyun.campus.client.FSOutputStream;
import com.ksyun.campus.client.domain.ClusterInfo;
import com.ksyun.campus.client.domain.DataServerMsg;
import com.ksyun.campus.client.domain.MetaServerMsg;
import com.ksyun.campus.client.domain.StatInfo;
import org.springframework.stereotype.Service;
import org.springframework.web.multipart.MultipartFile;

import java.io.ByteArrayOutputStream;
import java.io.IOException;
import java.io.InputStream;
import java.util.List;

/**
 * MinFS文件系统服务类
 * 封装EFileSystem的核心功能，提供给Web层调用
 */
@Service
public class MinFSService {

    private final EFileSystem fileSystem;

    public MinFSService() {
        // 初始化文件系统，使用默认命名空间
        this.fileSystem = new EFileSystem("default");
    }

    /**
     * 创建目录
     */
    public boolean createDirectory(String path) {
        try {
            return fileSystem.mkdir(path);
        } catch (Exception e) {
            throw new RuntimeException("创建目录失败: " + e.getMessage(), e);
        }
    }

    /**
     * 上传文件
     */
    public boolean uploadFile(String path, MultipartFile file) {
        try (FSOutputStream outputStream = fileSystem.create(path);
             InputStream inputStream = file.getInputStream()) {
            
            // 增大缓冲区以提高大文件传输效率 (64KB)
            byte[] buffer = new byte[65536];
            int bytesRead;
            long totalBytes = 0;
            long fileSize = file.getSize();
            
            while ((bytesRead = inputStream.read(buffer)) != -1) {
                outputStream.write(buffer, 0, bytesRead);
                totalBytes += bytesRead;
                
                // 每传输10MB输出一次进度（仅用于大文件）
                if (fileSize > 10 * 1024 * 1024 && totalBytes % (10 * 1024 * 1024) == 0) {
                    double progress = (totalBytes * 100.0) / fileSize;
                    System.out.printf("上传进度: %.1f%% (%d/%d bytes)\n", 
                                    progress, totalBytes, fileSize);
                }
            }
            
            System.out.println("文件上传完成，总共传输: " + totalBytes + " bytes");
            return true;
        } catch (IOException e) {
            throw new RuntimeException("上传文件失败: " + e.getMessage(), e);
        }
    }

    /**
     * 上传文件（字节数组）
     */
    public boolean uploadFile(String path, byte[] data) {
        try (FSOutputStream outputStream = fileSystem.create(path)) {
            outputStream.write(data);
            return true;
        } catch (IOException e) {
            throw new RuntimeException("上传文件失败: " + e.getMessage(), e);
        }
    }

    /**
     * 下载文件
     */
    public byte[] downloadFile(String path) {
        try (FSInputStream inputStream = fileSystem.open(path);
             ByteArrayOutputStream outputStream = new ByteArrayOutputStream()) {
            
            byte[] buffer = new byte[8192];
            int bytesRead;
            while ((bytesRead = inputStream.read(buffer)) != -1) {
                outputStream.write(buffer, 0, bytesRead);
            }
            return outputStream.toByteArray();
        } catch (IOException e) {
            throw new RuntimeException("下载文件失败: " + e.getMessage(), e);
        }
    }

    /**
     * 删除文件或目录
     */
    public boolean delete(String path) {
        try {
            return fileSystem.delete(path);
        } catch (Exception e) {
            throw new RuntimeException("删除失败: " + e.getMessage(), e);
        }
    }

    /**
     * 获取文件或目录信息
     */
    public StatInfo getFileInfo(String path) {
        try {
            return fileSystem.getFileStats(path);
        } catch (Exception e) {
            throw new RuntimeException("获取文件信息失败: " + e.getMessage(), e);
        }
    }

    /**
     * 列出目录内容
     */
    public List<StatInfo> listDirectory(String path) {
        try {
            return fileSystem.listFileStats(path);
        } catch (Exception e) {
            throw new RuntimeException("列出目录内容失败: " + e.getMessage(), e);
        }
    }

    /**
     * 获取集群信息
     */
    public ClusterInfo getClusterInfo() {
        try {
            return fileSystem.getClusterInfo();
        } catch (Exception e) {
            throw new RuntimeException("获取集群信息失败: " + e.getMessage(), e);
        }
    }

    /**
     * 检查文件或目录是否存在
     */
    public boolean exists(String path) {
        try {
            StatInfo info = fileSystem.getFileStats(path);
            return info != null;
        } catch (Exception e) {
            return false;
        }
    }

    /**
     * 获取文件大小
     */
    public long getFileSize(String path) {
        try {
            StatInfo info = fileSystem.getFileStats(path);
            return info != null ? info.getSize() : -1;
        } catch (Exception e) {
            return -1;
        }
    }

    /**
     * 执行服务器控制命令 - 基于集群信息动态获取IP和端口
     */
    public synchronized String executeServerCommand(String serverType, String action, String serverId) {
        try {
            String targetHost = null;
            String targetPort = null;
            
            // 对于所有操作，使用固定的端口映射规则
            if ("stop".equals(action) || "start".equals(action) || "restart".equals(action) || "status".equals(action)) {
                targetHost = "localhost";  // 默认使用localhost
                
                // 前端现在直接传递端口号，不需要再次计算
                if ("meta".equals(serverType)) {
                    // 前端传递的serverId现在是端口号（如"9090", "9091", "9092"）
                    targetPort = serverId != null && serverId.matches("\\d+") ? serverId : "9090";
                } else if ("data".equals(serverType)) {
                    // 前端传递的serverId现在是端口号（如"8001", "8002", "8003", "8004"）
                    targetPort = serverId != null && serverId.matches("\\d+") ? serverId : "8001";
                }
            } else {
                // 对于启动和状态查询，尝试获取集群信息
                try {
                    ClusterInfo clusterInfo = getClusterInfo();
                    
                    if ("meta".equals(serverType)) {
                        // 处理MetaServer
                        if ("master".equals(serverId) && clusterInfo.getMasterMetaServer() != null) {
                            MetaServerMsg master = clusterInfo.getMasterMetaServer();
                            targetHost = master.getHost();
                            targetPort = String.valueOf(master.getPort());
                        } else if (clusterInfo.getSlaveMetaServer() != null) {
                            // 处理slave server，serverId可能是索引
                            try {
                                int index = Integer.parseInt(serverId);
                                if (index >= 0 && index < clusterInfo.getSlaveMetaServer().size()) {
                                    MetaServerMsg slave = clusterInfo.getSlaveMetaServer().get(index);
                                    targetHost = slave.getHost();
                                    targetPort = String.valueOf(slave.getPort());
                                }
                            } catch (NumberFormatException e) {
                                // 如果不是数字索引，使用传统模式
                                targetHost = "localhost";
                                targetPort = serverId;
                            }
                        }
                    } else if ("data".equals(serverType)) {
                        // 处理DataServer
                        if (clusterInfo.getDataServer() != null) {
                            try {
                                int index = Integer.parseInt(serverId);
                                if (index >= 0 && index < clusterInfo.getDataServer().size()) {
                                    DataServerMsg dataServer = clusterInfo.getDataServer().get(index);
                                    targetHost = dataServer.getHost();
                                    targetPort = String.valueOf(dataServer.getPort());
                                }
                            } catch (NumberFormatException e) {
                                // 如果不是数字索引，使用传统模式
                                targetHost = "localhost";
                                targetPort = serverId;
                            }
                        }
                    }
                } catch (Exception e) {
                    // 集群信息获取失败时的fallback
                    System.err.println("获取集群信息失败，使用默认配置: " + e.getMessage());
                    targetHost = "localhost";
                    if ("meta".equals(serverType)) {
                        targetPort = serverId.matches("\\d+") ? serverId : "9090";
                    } else {
                        targetPort = serverId.matches("\\d+") ? serverId : "8001";
                    }
                }
            }
            
            if (targetHost == null || targetPort == null) {
                return "无法确定服务器地址: " + serverType + " " + serverId;
            }
            
            // 执行脚本，传递真实的IP和端口
            String scriptPath = getScriptPath(serverType, action);
            // 执行脚本，传递主机和端口参数
            ProcessBuilder processBuilder = new ProcessBuilder("bash", scriptPath, targetHost, targetPort);
            processBuilder.redirectErrorStream(true);
            Process process = processBuilder.start();
            
            // 读取命令输出
            StringBuilder output = new StringBuilder();
            try (java.io.BufferedReader reader = new java.io.BufferedReader(
                    new java.io.InputStreamReader(process.getInputStream()))) {
                String line;
                while ((line = reader.readLine()) != null) {
                    output.append(line).append("\n");
                }
            }
            
            int exitCode = process.waitFor();
            String result = output.toString().trim();
            
            if (exitCode == 0) {
                return result.isEmpty() ? "操作成功完成" : result;
            } else {
                throw new RuntimeException("脚本执行失败 (退出码: " + exitCode + "): " + result);
            }
            
        } catch (Exception e) {
            throw new RuntimeException("执行服务器控制命令失败: " + e.getMessage(), e);
        }
    }

    /**
     * 获取脚本路径
     */
    private String getScriptPath(String serverType, String action) {
        String scriptDir = System.getProperty("user.dir") + "/scripts/";
        
        switch (serverType.toLowerCase()) {
            case "meta":
                switch (action.toLowerCase()) {
                    case "start": return scriptDir + "start_metaServer.sh";
                    case "stop": return scriptDir + "stop_metaServer.sh";
                    case "restart": return scriptDir + "restart_metaServer.sh";
                    case "status": return scriptDir + "status_metaServer.sh";
                    default: throw new IllegalArgumentException("不支持的MetaServer操作: " + action);
                }
            case "data":
                switch (action.toLowerCase()) {
                    case "start": return scriptDir + "start_dataServer.sh";
                    case "stop": return scriptDir + "stop_dataServer.sh";
                    case "restart": return scriptDir + "restart_dataServer.sh";
                    case "status": return scriptDir + "status_dataServer.sh";
                    default: throw new IllegalArgumentException("不支持的DataServer操作: " + action);
                }
            default:
                throw new IllegalArgumentException("不支持的服务器类型: " + serverType);
        }
    }
}
