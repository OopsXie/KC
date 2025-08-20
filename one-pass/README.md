# One-Pass Payment System

一个基于Go语言开发的高性能支付系统，支持批量支付、用户交易和余额查询等核心功能。系统采用微服务架构设计，具备高并发处理能力和完善的错误处理机制。

## 项目概述

One-Pass Payment System是一个企业级支付解决方案，专为处理大规模并发交易而设计。系统通过Redis缓存和MySQL数据库提供高性能的数据存储，采用分布式锁机制确保数据一致性，支持动态并发控制和智能重试策略。

## 技术栈

- **编程语言**: Go 1.24.1
- **Web框架**: Gin Framework
- **数据库**: MySQL 8.0+ with GORM ORM
- **缓存系统**: Redis 6.0+
- **配置管理**: Viper
- **HTTP客户端**: 标准库net/http
- **UUID生成**: Google UUID
- **精度计算**: shopspring/decimal
- **容器化**: Docker & Docker Compose

## 核心功能

### 1. 批量支付处理
- 支持大批量用户的并发支付操作
- 智能并发控制，根据处理成功率动态调整并发数
- 分阶段处理策略（快速阶段 + 慢速阶段）
- 完善的错误处理和重试机制
- Redis分布式锁防止重复处理

### 2. 用户交易管理
- 支持用户间余额转账
- 高精度金额计算，避免浮点数精度问题
- 原子性事务处理，确保数据一致性
- 支持字符串和数字两种金额格式输入
- 实时余额更新和缓存同步

### 3. 余额查询服务
- 批量用户余额查询
- Redis缓存优化，提升查询性能
- 缓存与数据库双重保障
- 灵活的查询参数支持

### 4. 系统监控与健康检查
- 内置健康检查接口
- Redis缓存初始化功能
- 余额验证和修复工具
- 详细的日志记录

## 系统架构

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   API Gateway   │    │   Middleware    │    │    Handler      │
│                 │    │                 │    │                 │
│ • Rate Limit    │────│ • Logger        │────│ • Batch Pay     │
│ • Concurrency   │    │ • Recovery      │    │ • User Trade    │
│   Control       │    │ • CORS          │    │ • Query         │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                                        │
                       ┌─────────────────┐             │
                       │    Service      │             │
                       │                 │◄────────────┘
                       │ • Batch Service │
                       │ • Trade Service │
                       │ • Query Service │
                       └─────────────────┘
                                │
               ┌────────────────┼────────────────┐
               │                │                │
    ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐
    │     MySQL       │ │     Redis       │ │   External API  │
    │                 │ │                 │ │                 │
    │ • User Balance  │ │ • Cache Layer   │ │ • Third Party   │
    │ • Transaction   │ │ • Distributed   │ │   Payment API   │
    │   Log           │ │   Lock          │ │                 │
    └─────────────────┘ └─────────────────┘ └─────────────────┘
```

## API接口

### 1. 批量支付接口
```http
POST /onePass/batchPay
Content-Type: application/json

{
    "batchPayId": "batch_20240723_001",
    "uids": [1001, 1002, 1003, 1004, 1005]
}
```

**响应示例:**
```json
{
    "msg": "ok",
    "code": 200,
    "requestId": "uuid-string",
    "data": null
}
```

### 2. 用户交易接口
```http
POST /onePass/userTrade
Content-Type: application/json

{
    "sourceUid": 1001,
    "targetUid": 1002,
    "amount": "100.50"
}
```

**响应示例:**
```json
{
    "code": 200,
    "msg": "交易成功",
    "requestId": "uuid-string",
    "data": {
        "sourceUid": 1001,
        "targetUid": 1002,
        "amount": 100.50,
        "timestamp": 1642764123
    }
}
```

### 3. 用户余额查询接口
```http
POST /onePass/queryUserAmount
Content-Type: application/json

[1001, 1002, 1003]
```

**响应示例:**
```json
{
    "code": 200,
    "msg": "ok",
    "requestId": "uuid-string",
    "data": [
        {
            "uid": 1001,
            "amount": 50000.00
        },
        {
            "uid": 1002,
            "amount": 30000.00
        }
    ]
}
```

### 4. 系统健康检查
```http
GET /health
```

**响应示例:**
```json
{
    "status": "healthy",
    "service": "one-pass",
    "timestamp": 1642764123
}
```

### 5. Redis缓存初始化
```http
POST /onePass/initRedisCache
```

### 6. 余额验证接口
```http
POST /onePass/validateBalance
Content-Type: application/json

{
    "uid": 1001,
    "expectedAmount": 50000.00
}
```

## 配置说明

系统使用YAML格式的配置文件，位于`config/config.yaml`：

```yaml
kingstar:
  id: "40008"
  token: "Xzh000LyPq7Wm1Ae"

server:
  ip: "172.16.0.7"
  baseUrl: "http://172.16.0.7:40008"

gitlab:
  url: "http://120.92.88.48/whkj_xiezihang/one-pass.git"

# API接口配置
api:
  payUrl: "http://172.16.0.93/thirdpart/onePass/pay"
  batchPayBeginUrl: "http://172.16.0.93/thirdpart/onePass/batchPayBegin"
  batchPayFinishUrl: "http://172.16.0.93/thirdpart/onePass/batchPayFinish"

# 数据库配置
mysql:
  dsn: "root:root@tcp(mysql:3306)/onepass?charset=utf8mb4&parseTime=True&loc=Local"

# Redis配置
redis:
  addr: "redis:6379"
  password: ""
  db: 0
```

## 数据库设计

### 用户余额表 (user_balances)
```sql
CREATE TABLE user_balances (
    uid BIGINT PRIMARY KEY,
    amount DECIMAL(15,2) NOT NULL DEFAULT 0.00,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_uid (uid)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

## 性能优化

### 数据库优化
- **连接池配置**: 最大连接数100，空闲连接数20
- **索引优化**: 为用户ID添加数据库索引
- **事务控制**: 使用行级锁确保数据一致性
- **连接复用**: 配置连接最大生存时间和空闲时间

### 缓存策略
- **Redis分布式锁**: 防止并发冲突
- **批量操作结果缓存**: 减少数据库压力
- **连接池复用**: 降低网络开销
- **缓存过期策略**: 24小时自动过期

### 并发控制
- **全局并发限制**: 最大同时处理500个请求
- **速率限制**: 每秒最多100个请求，基于客户端IP
- **动态并发调整**: 根据成功率自动调整并发数(10-60之间)
- **分阶段处理**: 快速阶段(30秒) + 慢速阶段处理

### 中间件系统
- **并发限制中间件**: 使用信号量控制最大并发数
- **速率限制中间件**: 基于滑动窗口算法的请求频率控制
- **恢复中间件**: 自动处理panic异常
- **日志中间件**: 结构化请求日志记录

## 部署指南

### 环境要求
- Go 1.24.1+
- Docker & Docker Compose
- MySQL 8.0+
- Redis 6.0+

### Docker部署

1. **克隆项目**
```bash
git clone http://120.92.88.48/whkj_xiezihang/one-pass.git
cd one-pass
```

2. **构建并启动服务**
```bash
docker-compose up -d --build
```

3. **验证部署**
```bash
# 检查容器状态
docker-compose ps

# 测试健康检查
curl http://172.16.0.7:40008/health

# 查看应用日志
docker-compose logs -f onepass-app
```

### 端口映射
- **应用服务**: 172.16.0.7:40008
- **MySQL**: 172.16.0.7:8002
- **Redis**: 172.16.0.7:7002

### 本地开发

1. **安装依赖**
```bash
go mod download
```

2. **启动MySQL和Redis**
```bash
docker-compose up -d mysql redis
```

3. **运行应用**
```bash
go run main.go
```

## 监控与维护

### 日志管理
```bash
# 查看应用日志
docker-compose logs -f onepass-app

# 查看数据库日志
docker-compose logs -f mysql

# 查看Redis日志
docker-compose logs -f redis
```

### 数据备份
```bash
# MySQL数据备份
docker-compose exec mysql mysqldump -u root -proot onepass > backup.sql

# Redis数据备份
docker-compose exec redis redis-cli BGSAVE
```

### 服务管理
```bash
# 重启服务
docker-compose restart onepass-app

# 扩展服务（如果支持）
docker-compose up -d --scale onepass-app=3

# 查看资源使用情况
docker stats
```

## 错误处理

系统实现了完善的错误处理机制：

- **业务错误**: 返回具体的错误码和错误信息
- **系统错误**: 自动重试和降级处理
- **网络错误**: 连接超时和重试机制
- **数据一致性**: 事务回滚和补偿机制

## 安全考虑

- **输入验证**: 严格的参数校验和类型检查
- **SQL注入防护**: 使用GORM ORM框架防止SQL注入
- **并发控制**: 分布式锁防止竞态条件
- **精度控制**: 使用decimal库避免浮点数精度问题

## 开发指南

### 项目结构
```
one-pass/
├── main.go                 # 应用入口
├── config/                 # 配置管理
│   ├── config.go
│   └── config.yaml
├── handler/                # HTTP处理器
│   ├── batch_pay.go
│   ├── query_handler.go
│   └── trade_handler.go
├── service/                # 业务逻辑层
│   ├── batch_service.go
│   ├── query_service.go
│   └── trade_service.go
├── model/                  # 数据模型
│   ├── user_balance.go
│   ├── trade.go
│   ├── query.go
│   └── request.go
├── middleware/             # 中间件
│   ├── rate_limit.go
│   └── logger.go
├── utils/                  # 工具函数
│   └── http.go
├── docker-compose.yml      # Docker编排文件
├── Dockerfile             # Docker构建文件
└── README.md              # 项目文档
```

### 代码规范
- 使用Go官方代码规范
- 函数和方法必须包含注释
- 错误处理必须明确和完善
- 使用结构化日志记录
- 单元测试覆盖核心业务逻辑

## 故障排查

### 常见问题

1. **数据库连接失败**
   - 检查MySQL服务状态
   - 验证连接字符串配置
   - 查看网络连通性

2. **Redis连接异常**
   - 检查Redis服务状态
   - 验证Redis配置参数
   - 查看内存使用情况

3. **高并发性能问题**
   - 监控数据库连接池状态
   - 检查Redis缓存命中率
   - 调整并发控制参数

4. **交易数据不一致**
   - 检查分布式锁状态
   - 验证事务回滚机制
   - 查看错误重试日志
