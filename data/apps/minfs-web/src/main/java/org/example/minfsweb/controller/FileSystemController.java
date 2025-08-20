package org.example.minfsweb.controller;

import com.ksyun.campus.client.domain.StatInfo;
import com.ksyun.campus.client.domain.FileType;

import org.example.minfsweb.dto.ClusterInfoDTO;
import org.example.minfsweb.dto.FileInfoDTO;
import org.example.minfsweb.result.Result;
import org.example.minfsweb.service.MinFSService;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.HttpHeaders;
import org.springframework.http.HttpStatus;
import org.springframework.http.MediaType;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;
import org.springframework.web.multipart.MultipartFile;

import java.util.List;
import java.util.stream.Collectors;

/**
 * MinFS文件系统控制器
 * 提供文件和目录操作的RESTful API
 */
@RestController
@RequestMapping("/api/fs")
@CrossOrigin(origins = "*")
public class FileSystemController {

    @Autowired
    private MinFSService minFSService;

    /**
     * 创建目录
     */
    @PostMapping("/mkdir")
    public Result<Boolean> createDirectory(@RequestParam String path) {
        try {
            boolean success = minFSService.createDirectory(path);
            if (success) {
                Result<Boolean> result = Result.success(true);
                result.setMsg("目录创建成功");
                return result;
            } else {
                return Result.error("目录创建失败");
            }
        } catch (Exception e) {
            return Result.error("创建目录时发生错误: " + e.getMessage());
        }
    }

    /**
     * 上传文件
     */
    @PostMapping("/upload")
    public Result<Boolean> uploadFile(@RequestParam String path, 
                                     @RequestParam("file") MultipartFile file) {
        try {
            // 检查文件是否为空
            if (file.isEmpty()) {
                return Result.error("上传文件不能为空");
            }
            
            // 检查文件大小
            long fileSize = file.getSize();
            long maxSize = 500L * 1024L * 1024L; // 500MB
            if (fileSize > maxSize) {
                return Result.error("文件大小超过限制（最大500MB），当前文件大小: " + formatFileSize(fileSize));
            }
            
            // 记录上传开始
            String fileName = file.getOriginalFilename();
            System.out.println("开始上传文件: " + fileName + ", 大小: " + formatFileSize(fileSize) + ", 目标路径: " + path);
            
            long startTime = System.currentTimeMillis();
            boolean success = minFSService.uploadFile(path, file);
            long endTime = System.currentTimeMillis();
            
            if (success) {
                String uploadTime = String.format("%.2f", (endTime - startTime) / 1000.0);
                String successMsg = String.format("文件上传成功 (大小: %s, 耗时: %s秒)", 
                                                 formatFileSize(fileSize), uploadTime);
                System.out.println(successMsg);
                
                Result<Boolean> result = Result.success(true);
                result.setMsg(successMsg);
                return result;
            } else {
                return Result.error("文件上传失败");
            }
        } catch (Exception e) {
            String errorMsg = "上传文件时发生错误: " + e.getMessage();
            System.err.println(errorMsg);
            e.printStackTrace();
            return Result.error(errorMsg);
        }
    }
    
    /**
     * 格式化文件大小
     */
    private String formatFileSize(long size) {
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
     * 下载文件
     */
    @GetMapping("/download")
    public ResponseEntity<byte[]> downloadFile(@RequestParam String path) {
        try {
            byte[] fileData = minFSService.downloadFile(path);
            
            // 从路径中提取文件名
            String filename = path.substring(path.lastIndexOf("/") + 1);
            
            HttpHeaders headers = new HttpHeaders();
            headers.setContentType(MediaType.APPLICATION_OCTET_STREAM);
            headers.setContentDispositionFormData("attachment", filename);
            headers.setContentLength(fileData.length);
            
            return new ResponseEntity<>(fileData, headers, HttpStatus.OK);
        } catch (Exception e) {
            return new ResponseEntity<>(HttpStatus.INTERNAL_SERVER_ERROR);
        }
    }

    /**
     * 删除文件或目录
     */
    @DeleteMapping("/delete")
    public Result<Boolean> delete(@RequestParam String path) {
        try {
            boolean success = minFSService.delete(path);
            if (success) {
                Result<Boolean> result = Result.success(true);
                result.setMsg("删除成功");
                return result;
            } else {
                return Result.error("删除失败");
            }
        } catch (Exception e) {
            return Result.error("删除时发生错误: " + e.getMessage());
        }
    }

    /**
     * 获取文件或目录信息
     */
    @GetMapping("/info")
    public Result<FileInfoDTO> getFileInfo(@RequestParam String path) {
        try {
            StatInfo statInfo = minFSService.getFileInfo(path);
            if (statInfo == null) {
                return Result.error("文件或目录不存在");
            }
            
            // 只对文件类型获取副本信息，目录类型不包含副本信息
            List<com.ksyun.campus.client.domain.ReplicaData> replicaData = null;
            if (statInfo.getType() == FileType.File) {
                replicaData = statInfo.getReplicaData();
            }
            
            FileInfoDTO dto = new FileInfoDTO(statInfo.getPath(), statInfo.getSize(), 
                                            statInfo.getMtime(), statInfo.getType(), 
                                            replicaData);
            return Result.success(dto);
        } catch (Exception e) {
            return Result.error("获取文件信息时发生错误: " + e.getMessage());
        }
    }

    /**
     * 列出目录内容
     */
    @GetMapping("/list")
    public Result<List<FileInfoDTO>> listDirectory(@RequestParam String path) {
        try {
            List<StatInfo> statInfos = minFSService.listDirectory(path);
            List<FileInfoDTO> dtos = statInfos.stream()
                    .map(info -> {
                        // 只对文件类型获取副本信息，目录类型不包含副本信息
                        List<com.ksyun.campus.client.domain.ReplicaData> replicaData = null;
                        if (info.getType() == FileType.File) {
                            replicaData = info.getReplicaData();
                        }
                        return new FileInfoDTO(info.getPath(), info.getSize(), 
                                             info.getMtime(), info.getType(), 
                                             replicaData);
                    })
                    .collect(Collectors.toList());
            return Result.success(dtos);
        } catch (Exception e) {
            return Result.error("列出目录内容时发生错误: " + e.getMessage());
        }
    }

    /**
     * 检查文件或目录是否存在
     */
    @GetMapping("/exists")
    public Result<Boolean> exists(@RequestParam String path) {
        try {
            boolean exists = minFSService.exists(path);
            return Result.success(exists);
        } catch (Exception e) {
            return Result.error("检查文件存在性时发生错误: " + e.getMessage());
        }
    }

    /**
     * 获取文件大小
     */
    @GetMapping("/size")
    public Result<Long> getFileSize(@RequestParam String path) {
        try {
            long size = minFSService.getFileSize(path);
            return size >= 0 ? Result.success(size) 
                             : Result.error("文件不存在或无法获取大小");
        } catch (Exception e) {
            return Result.error("获取文件大小时发生错误: " + e.getMessage());
        }
    }

    /**
     * 获取集群信息
     */
    @GetMapping("/cluster")
    public Result<ClusterInfoDTO> getClusterInfo() {
        try {
            ClusterInfoDTO clusterInfo = new ClusterInfoDTO(minFSService.getClusterInfo());
            return Result.success(clusterInfo);
        } catch (Exception e) {
            return Result.error("获取集群信息时发生错误: " + e.getMessage());
        }
    }

    /**
     * 健康检查接口
     */
    @GetMapping("/health")
    public Result<String> health() {
        try {
            // 尝试获取根目录信息来检查连接
            minFSService.exists("/");
            return Result.success("MinFS连接正常");
        } catch (Exception e) {
            return Result.error("MinFS连接异常: " + e.getMessage());
        }
    }

    /**
     * 启动MetaServer
     */
    @PostMapping("/server/meta/start")
    public Result<String> startMetaServer(@RequestParam(required = false) String serverId) {
        try {
            String result = minFSService.executeServerCommand("meta", "start", serverId);
            return Result.success(result);
        } catch (Exception e) {
            return Result.error("启动MetaServer失败: " + e.getMessage());
        }
    }

    /**
     * 停止MetaServer
     */
    @PostMapping("/server/meta/stop")
    public Result<String> stopMetaServer(@RequestParam(required = false) String serverId) {
        try {
            String result = minFSService.executeServerCommand("meta", "stop", serverId);
            return Result.success(result);
        } catch (Exception e) {
            return Result.error("停止MetaServer失败: " + e.getMessage());
        }
    }

    /**
     * 启动DataServer
     */
    @PostMapping("/server/data/start")
    public Result<String> startDataServer(@RequestParam(required = false) String serverId) {
        try {
            String result = minFSService.executeServerCommand("data", "start", serverId);
            return Result.success(result);
        } catch (Exception e) {
            return Result.error("启动DataServer失败: " + e.getMessage());
        }
    }

    /**
     * 停止DataServer
     */
    @PostMapping("/server/data/stop")
    public Result<String> stopDataServer(@RequestParam(required = false) String serverId) {
        try {
            String result = minFSService.executeServerCommand("data", "stop", serverId);
            return Result.success(result);
        } catch (Exception e) {
            return Result.error("停止DataServer失败: " + e.getMessage());
        }
    }

    /**
     * 重启服务器
     */
    @PostMapping("/server/restart")
    public Result<String> restartServer(@RequestParam String serverType, 
                                       @RequestParam(required = false) String serverId) {
        try {
            String result = minFSService.executeServerCommand(serverType, "restart", serverId);
            return Result.success(result);
        } catch (Exception e) {
            return Result.error("重启服务器失败: " + e.getMessage());
        }
    }

    /**
     * 获取服务器状态
     */
    @GetMapping("/server/status")
    public Result<String> getServerStatus(@RequestParam String serverType,
                                         @RequestParam(required = false) String serverId) {
        try {
            String result = minFSService.executeServerCommand(serverType, "status", serverId);
            return Result.success(result);
        } catch (Exception e) {
            return Result.error("获取服务器状态失败: " + e.getMessage());
        }
    }
}
