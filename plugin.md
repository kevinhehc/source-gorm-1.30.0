# GORM 开源插件大全（官方 + 社区）

截至 2025 年，GORM 官方与社区提供了多个开源插件，涵盖了数据库分离、审计、链路追踪、软删除、加密、缓存等方面。以下是整理清单：

---

## ✅ 一、GORM 官方插件（gorm.io 官方发布）

| 插件名称 | 简介 | 安装路径 |
|----------|------|-----------|
| **dbresolver** | 读写分离、主从数据库支持 | `gorm.io/plugin/dbresolver` |
| **soft_delete** | 支持软删除字段的统一处理 | `gorm.io/plugin/soft_delete` |
| **prometheus** | Prometheus 监控指标集成 | `gorm.io/plugin/prometheus` |
| **opentelemetry** | OpenTelemetry 链路追踪 | `gorm.io/plugin/opentelemetry/tracing` |
| **tracing** | 简化版 tracing 插件，用于 Zipkin/Jaeger 等 | `gorm.io/plugin/opentelemetry/tracing` |
| **dbresolver-cockroachdb** | CockroachDB 的主从复制支持 | `gorm.io/plugin/dbresolver/cockroachdb` |

---

## 🌱 二、社区流行插件（广泛使用，非官方）

### 🔐 安全 / 加密相关

| 插件 | 功能 | GitHub |
|------|------|--------|
| **gorm-crypto** | 字段级加密/解密（使用 AES、RSA） | [tkuchiki/gorm-crypto](https://github.com/tkuchiki/gorm-crypto) |
| **gorm-encrypted** | 多算法字段加密（支持 AES、Hash 等） | [rogchap/gorm-encrypted](https://github.com/rogchap/gorm-encrypted) |

### 📝 审计 / 多租户

| 插件 | 功能 | GitHub |
|------|------|--------|
| **gorm-audit** | 自动记录 `created_by`、`updated_by` 字段 | [sunary/gorm-audit](https://github.com/sunary/gorm-audit) |
| **gorm-tenant** | 多租户租户隔离插件 | [go-gorm/gorm-tenant](https://github.com/go-gorm/gorm-tenant) |

### ⚡ 缓存 / 性能优化

| 插件 | 功能 | GitHub |
|------|------|--------|
| **gorm-cache** | 查询缓存（支持 Redis、本地缓存） | [eko/gorm-cache](https://github.com/eko/gorm-cache) |
| **go-gorm/cache** | 官方实验缓存插件（已不活跃） | [go-gorm/cache](https://github.com/go-gorm/cache) |

### 🧪 测试 / Mock 工具

| 插件 | 功能 | GitHub |
|------|------|--------|
| **gorm-mock** | GORM 接口 mock，用于测试 | [DATA-DOG/go-sqlmock](https://github.com/DATA-DOG/go-sqlmock) |

### 🛠 工具性插件

| 插件 | 功能 | GitHub |
|------|------|--------|
| **gorm-logger** | 自定义日志输出（如支持 logrus） | [onrik/gorm-logrus](https://github.com/onrik/gorm-logrus) |
| **gorm-gen** | 模型和 CURD 代码生成器（官方工具） | [gorm.io/gen](https://gorm.io/gen/) |
| **gormigrate** | 数据库版本迁移工具 | [go-gormigrate/gormigrate](https://github.com/go-gormigrate/gormigrate) |

---

## 📦 插件安装方式示例（以 dbresolver 为例）

```go
import (
    "gorm.io/gorm"
    "gorm.io/plugin/dbresolver"
)

db.Use(dbresolver.Register(dbresolver.Config{
    Sources:  []gorm.Dialector{mysql.Open("master_dsn")},
    Replicas: []gorm.Dialector{mysql.Open("replica_dsn")},
}))