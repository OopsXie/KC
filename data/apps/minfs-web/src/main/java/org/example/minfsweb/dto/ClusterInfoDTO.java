package org.example.minfsweb.dto;



import com.ksyun.campus.client.domain.ClusterInfo;
import com.ksyun.campus.client.domain.DataServerMsg;
import com.ksyun.campus.client.domain.MetaServerMsg;

import java.util.List;

/**
 * 集群信息DTO
 */
public class ClusterInfoDTO {
    private MetaServerInfo masterMetaServer;
    private List<MetaServerInfo> slaveMetaServers;
    private List<DataServerInfo> dataServers;
    private int totalMetaServers;
    private int totalDataServers;

    public ClusterInfoDTO() {}

    public ClusterInfoDTO(ClusterInfo clusterInfo) {
        if (clusterInfo.getMasterMetaServer() != null) {
            this.masterMetaServer = new MetaServerInfo(clusterInfo.getMasterMetaServer());
        }
        
        if (clusterInfo.getSlaveMetaServer() != null) {
            this.slaveMetaServers = clusterInfo.getSlaveMetaServer().stream()
                    .map(MetaServerInfo::new)
                    .collect(java.util.stream.Collectors.toList());
            this.totalMetaServers = this.slaveMetaServers.size() + (masterMetaServer != null ? 1 : 0);
        }
        
        if (clusterInfo.getDataServer() != null) {
            this.dataServers = clusterInfo.getDataServer().stream()
                    .map(DataServerInfo::new)
                    .collect(java.util.stream.Collectors.toList());
            this.totalDataServers = this.dataServers.size();
        }
    }

    // 内部类：MetaServer信息
    public static class MetaServerInfo {
        private String address;
        private String status;
        private String host;
        private int port;

        public MetaServerInfo() {}

        public MetaServerInfo(MetaServerMsg metaServerMsg) {
            // 使用真实的MetaServerMsg字段进行映射
            if (metaServerMsg != null) {
                this.host = metaServerMsg.getHost();
                this.port = metaServerMsg.getPort();
                this.address = this.host + ":" + this.port;
                this.status = "Active"; // 默认为活跃状态
            } else {
                this.address = "Unknown MetaServer";
                this.status = "Unknown";
                this.host = "unknown";
                this.port = 0;
            }
        }

        // Getters and Setters
        public String getAddress() { return address; }
        public void setAddress(String address) { this.address = address; }
        public String getStatus() { return status; }
        public void setStatus(String status) { this.status = status; }
        public String getHost() { return host; }
        public void setHost(String host) { this.host = host; }
        public int getPort() { return port; }
        public void setPort(int port) { this.port = port; }
    }

    // 内部类：DataServer信息
    public static class DataServerInfo {
        private String address;
        private String status;
        private long capacity;
        private long used;
        private int fileTotal;  // 添加文件总数字段
        private String host;    // 添加主机字段
        private int port;       // 添加端口字段

        public DataServerInfo() {}

        public DataServerInfo(DataServerMsg dataServerMsg) {
            // 使用真实的DataServerMsg字段进行映射
            if (dataServerMsg != null) {
                // 基本信息
                this.host = dataServerMsg.getHost();
                this.port = dataServerMsg.getPort();
                this.address = this.host + ":" + this.port;
                this.status = "Active"; // 默认为活跃状态
                
                // 容量信息，保持原始的MB单位，前端会处理格式化
                this.capacity = dataServerMsg.getCapacity(); // 保持MB单位
                this.used = dataServerMsg.getUseCapacity();   // 保持MB单位
                
                // 文件总数（副本数）
                this.fileTotal = dataServerMsg.getFileTotal();
            } else {
                this.address = "Unknown DataServer";
                this.status = "Unknown";
                this.capacity = 0;
                this.used = 0;
                this.fileTotal = 0;
                this.host = "unknown";
                this.port = 0;
            }
        }


        // Getters and Setters
        public String getAddress() { return address; }
        public void setAddress(String address) { this.address = address; }
        public String getStatus() { return status; }
        public void setStatus(String status) { this.status = status; }
        public long getCapacity() { return capacity; }
        public void setCapacity(long capacity) { this.capacity = capacity; }
        public long getUsed() { return used; }
        public void setUsed(long used) { this.used = used; }
        public int getFileTotal() { return fileTotal; }
        public void setFileTotal(int fileTotal) { this.fileTotal = fileTotal; }
        public String getHost() { return host; }
        public void setHost(String host) { this.host = host; }
        public int getPort() { return port; }
        public void setPort(int port) { this.port = port; }

        public String getUsagePercentage() {
            if (capacity == 0) {
                return "0%";
            }
            return String.format("%.2f%%", (used * 100.0) / capacity);
        }
    }

    // Getters and Setters
    public MetaServerInfo getMasterMetaServer() { return masterMetaServer; }
    public void setMasterMetaServer(MetaServerInfo masterMetaServer) { this.masterMetaServer = masterMetaServer; }
    
    public List<MetaServerInfo> getSlaveMetaServers() { return slaveMetaServers; }
    public void setSlaveMetaServers(List<MetaServerInfo> slaveMetaServers) { this.slaveMetaServers = slaveMetaServers; }
    
    public List<DataServerInfo> getDataServers() { return dataServers; }
    public void setDataServers(List<DataServerInfo> dataServers) { this.dataServers = dataServers; }
    
    public int getTotalMetaServers() { return totalMetaServers; }
    public void setTotalMetaServers(int totalMetaServers) { this.totalMetaServers = totalMetaServers; }
    
    public int getTotalDataServers() { return totalDataServers; }
    public void setTotalDataServers(int totalDataServers) { this.totalDataServers = totalDataServers; }
}
