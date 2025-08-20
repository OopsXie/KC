# MinFS Web API 接口文档

基于client目录下的核心方法，我已经为你的项目创建了完整的RESTful API接口来调用MinFS文件系统功能。

## 项目结构

```
src/main/java/org/example/minfsweb/
├── service/
│   └── MinFSService.java          # 文件系统服务层，封装EFileSystem功能
├── controller/
│   └── FileSystemController.java  # REST API控制器
├── dto/
│   ├── FileInfoDTO.java           # 文件信息传输对象
│   └── ClusterInfoDTO.java        # 集群信息传输对象
└── result/
    └── Result.java                # 统一响应结果类
```

## API 接口列表

### 1. 目录操作

#### 创建目录
- **接口**: `POST /api/fs/mkdir`
- **参数**: `path` (String) - 目录路径
- **返回**: `Result<Boolean>`
- **示例**: 
  ```bash
  curl -X POST "http://localhost:8080/api/fs/mkdir?path=/test/newdir"
  ```

#### 列出目录内容
- **接口**: `GET /api/fs/list`
- **参数**: `path` (String) - 目录路径
- **返回**: `Result<List<FileInfoDTO>>`
- **示例**:
  ```bash
  curl "http://localhost:8080/api/fs/list?path=/"
  ```

### 2. 文件操作

#### 上传文件
- **接口**: `POST /api/fs/upload`
- **参数**: 
  - `path` (String) - 文件保存路径
  - `file` (MultipartFile) - 上传的文件
- **返回**: `Result<Boolean>`
- **示例**:
  ```bash
  curl -X POST -F "file=@localfile.txt" "http://localhost:8080/api/fs/upload?path=/test/uploadfile.txt"
  ```

#### 下载文件
- **接口**: `GET /api/fs/download`
- **参数**: `path` (String) - 文件路径
- **返回**: 文件二进制流
- **示例**:
  ```bash
  curl "http://localhost:8080/api/fs/download?path=/test/file.txt" -o downloaded_file.txt
  ```

#### 删除文件或目录
- **接口**: `DELETE /api/fs/delete`
- **参数**: `path` (String) - 文件或目录路径
- **返回**: `Result<Boolean>`
- **示例**:
  ```bash
  curl -X DELETE "http://localhost:8080/api/fs/delete?path=/test/file.txt"
  ```

### 3. 文件信息查询

#### 获取文件或目录信息
- **接口**: `GET /api/fs/info`
- **参数**: `path` (String) - 文件或目录路径
- **返回**: `Result<FileInfoDTO>`
- **示例**:
  ```bash
  curl "http://localhost:8080/api/fs/info?path=/test/file.txt"
  ```

#### 检查文件是否存在
- **接口**: `GET /api/fs/exists`
- **参数**: `path` (String) - 文件或目录路径
- **返回**: `Result<Boolean>`
- **示例**:
  ```bash
  curl "http://localhost:8080/api/fs/exists?path=/test/file.txt"
  ```

#### 获取文件大小
- **接口**: `GET /api/fs/size`
- **参数**: `path` (String) - 文件路径
- **返回**: `Result<Long>`
- **示例**:
  ```bash
  curl "http://localhost:8080/api/fs/size?path=/test/file.txt"
  ```

### 4. 系统信息

#### 获取集群信息
- **接口**: `GET /api/fs/cluster`
- **返回**: `Result<ClusterInfoDTO>`
- **示例**:
  ```bash
  curl "http://localhost:8080/api/fs/cluster"
  ```

#### 健康检查
- **接口**: `GET /api/fs/health`
- **返回**: `Result<String>`
- **示例**:
  ```bash
  curl "http://localhost:8080/api/fs/health"
  ```

## 数据传输对象

### FileInfoDTO
```json
{
  "path": "/test/file.txt",
  "size": 1024,
  "mtime": 1692096000,
  "type": "File",
  "typeName": "文件",
  "formattedSize": "1.00 KB",
  "formattedTime": "2023-08-15 16:00:00"
}
```

### ClusterInfoDTO
```json
{
  "masterMetaServer": {
    "address": "MetaServer",
    "status": "Active"
  },
  "slaveMetaServers": [...],
  "dataServers": [...],
  "totalMetaServers": 1,
  "totalDataServers": 3
}
```

### 统一响应格式 Result<T>
```json
{
  "code": 200,
  "msg": "ok",
  "requestId": "uuid-string",
  "data": {...}
}
```

## 核心功能映射

| 原始方法 (EFileSystem) | Web API接口 | 功能描述 |
|----------------------|-------------|----------|
| `mkdir(path)` | `POST /api/fs/mkdir` | 创建目录 |
| `create(path)` + `FSOutputStream` | `POST /api/fs/upload` | 创建并写入文件 |
| `open(path)` + `FSInputStream` | `GET /api/fs/download` | 读取文件 |
| `delete(path)` | `DELETE /api/fs/delete` | 删除文件或目录 |
| `getFileStats(path)` | `GET /api/fs/info` | 获取文件信息 |
| `listFileStats(path)` | `GET /api/fs/list` | 列出目录内容 |
| `getClusterInfo()` | `GET /api/fs/cluster` | 获取集群信息 |

## 使用说明

1. **启动应用**: 运行 `mvn spring-boot:run` 或直接运行 `MinfsWebApplication`
2. **访问接口**: 默认端口8080，所有接口都在 `/api/fs` 路径下
3. **跨域支持**: 已配置 `@CrossOrigin(origins = "*")`
4. **错误处理**: 所有接口都有统一的异常处理和错误返回

## 注意事项

1. 确保MinFS集群正常运行
2. 检查网络连接到etcd服务器 (http://10.212.217.58:2379)
3. 大文件上传可能需要调整Spring Boot的文件上传限制
4. 生产环境建议配置适当的跨域策略和安全认证
