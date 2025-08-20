package org.example.minfsweb.dto;


import com.ksyun.campus.client.domain.FileType;
import com.ksyun.campus.client.domain.ReplicaData;

import java.util.List;
import java.util.stream.Collectors;

/**
 * 文件信息DTO
 */
public class FileInfoDTO {
    private String path;
    private long size;
    private long mtime;
    private long uploadTime;  // 文件上传时间
    private String type;
    private String typeName;
    private List<ReplicaInfoDTO> replicaData;
    private int replicaCount;

    public FileInfoDTO() {}

    public FileInfoDTO(String path, long size, long mtime, FileType type) {
        this.path = path;
        this.size = size;
        this.mtime = mtime;
        this.uploadTime = mtime; // 使用文件修改时间作为上传时间
        this.type = type != null ? type.name() : "Unknown";
        this.typeName = getFileTypeName(type);
    }

    public FileInfoDTO(String path, long size, long mtime, FileType type, List<ReplicaData> replicaData) {
        this.path = path;
        this.size = size;
        this.mtime = mtime;
        this.uploadTime = mtime; // 使用文件修改时间作为上传时间
        this.type = type != null ? type.name() : "Unknown";
        this.typeName = getFileTypeName(type);
        
        if (replicaData != null) {
            this.replicaData = replicaData.stream()
                    .map(replica -> new ReplicaInfoDTO(replica.getId(), replica.getDsNode(), replica.getPath()))
                    .collect(Collectors.toList());
            this.replicaCount = replicaData.size();
        } else {
            this.replicaCount = 0;
        }
    }

    private String getFileTypeName(FileType type) {
        if (type == null) {
            return "未知";
        }
        switch (type) {
            case File:
                return "文件";
            case Directory:
                return "目录";
            case Volume:
                return "卷";
            default:
                return "未知";
        }
    }

    // Getters and Setters
    public String getPath() {
        return path;
    }

    public void setPath(String path) {
        this.path = path;
    }

    public long getSize() {
        return size;
    }

    public void setSize(long size) {
        this.size = size;
    }

    public long getMtime() {
        return mtime;
    }

    public void setMtime(long mtime) {
        this.mtime = mtime;
    }

    public long getUploadTime() {
        return uploadTime;
    }

    public void setUploadTime(long uploadTime) {
        this.uploadTime = uploadTime;
    }

    public String getType() {
        return type;
    }

    public void setType(String type) {
        this.type = type;
    }

    public String getTypeName() {
        return typeName;
    }

    public void setTypeName(String typeName) {
        this.typeName = typeName;
    }

    /**
     * 格式化文件大小
     */
    public String getFormattedSize() {
        if (size < 1024) {
            return size + " B";
        } else if (size < 1024 * 1024) {
            return String.format("%.2f KB", size / 1024.0);
        } else if (size < 1024 * 1024 * 1024) {
            return String.format("%.2f MB", size / (1024.0 * 1024.0));
        } else {
            return String.format("%.2f GB", size / (1024.0 * 1024.0 * 1024.0));
        }
    }

    /**
     * 格式化上传时间
     */
    public String getFormattedUploadTime() {
        return formatTimestamp(uploadTime);
    }

    /**
     * 格式化修改时间
     */
    public String getFormattedTime() {
        return formatTimestamp(mtime);
    }
    
    /**
     * 通用时间戳格式化方法
     */
    private String formatTimestamp(long timestamp) {
        try {
            if (timestamp == 0) {
                return "未知时间";
            }
            
            long timestampMs;
            
            // 根据时间戳大小判断单位并转换为毫秒
            if (timestamp > 1e15) {
                // 纳秒转毫秒
                timestampMs = timestamp / 1000000;
            } else if (timestamp > 1e12) {
                // 微秒转毫秒
                timestampMs = timestamp / 1000;
            } else if (timestamp > 1e10) {
                // 已经是毫秒
                timestampMs = timestamp;
            } else {
                // 标准Unix时间戳(秒)转毫秒
                timestampMs = timestamp * 1000;
            }
            
            java.util.Date date = new java.util.Date(timestampMs);
            
            // 检查日期是否合理(1970-2100年)
            java.util.Calendar cal = java.util.Calendar.getInstance();
            cal.setTime(date);
            int year = cal.get(java.util.Calendar.YEAR);
            
            if (year < 1970 || year > 2100) {
                return "时间错误(" + timestamp + ")";
            }
            
            java.text.SimpleDateFormat sdf = new java.text.SimpleDateFormat("yyyy-MM-dd HH:mm:ss");
            sdf.setTimeZone(java.util.TimeZone.getTimeZone("Asia/Shanghai"));
            return sdf.format(date);
        } catch (Exception e) {
            return "时间解析错误(" + timestamp + ")";
        }
    }

    public List<ReplicaInfoDTO> getReplicaData() {
        return replicaData;
    }

    public void setReplicaData(List<ReplicaInfoDTO> replicaData) {
        this.replicaData = replicaData;
    }

    public int getReplicaCount() {
        return replicaCount;
    }

    public void setReplicaCount(int replicaCount) {
        this.replicaCount = replicaCount;
    }

    /**
     * 副本信息DTO
     */
    public static class ReplicaInfoDTO {
        private String id;
        private String dsNode;  // DataServer节点，格式为ip:port
        private String path;

        public ReplicaInfoDTO() {}

        public ReplicaInfoDTO(String id, String dsNode, String path) {
            this.id = id;
            this.dsNode = dsNode;
            this.path = path;
        }

        public String getId() {
            return id;
        }

        public void setId(String id) {
            this.id = id;
        }

        public String getDsNode() {
            return dsNode;
        }

        public void setDsNode(String dsNode) {
            this.dsNode = dsNode;
        }

        public String getPath() {
            return path;
        }

        public void setPath(String path) {
            this.path = path;
        }

        /**
         * 获取DataServer主机地址
         */
        public String getHost() {
            if (dsNode != null && dsNode.contains(":")) {
                return dsNode.split(":")[0];
            }
            return dsNode;
        }

        /**
         * 获取DataServer端口
         */
        public String getPort() {
            if (dsNode != null && dsNode.contains(":")) {
                String[] parts = dsNode.split(":");
                if (parts.length > 1) {
                    return parts[1];
                }
            }
            return "";
        }
    }
}
