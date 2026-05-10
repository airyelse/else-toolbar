# Else Toolbar - 开发文档

基于 Wails v3 + Vue 3 的本地工具箱应用。

## 技术栈

| 层级 | 技术 |
|------|------|
| 框架 | Wails v3 |
| 前端 | Vue 3 + Element Plus + TypeScript |
| 后端 | Go 1.25+ |
| 数据库 | SQLite + GORM (pure-go, no CGO) |
| 加密 | AES-256-GCM |

## 项目结构

```
else-toolbar/
├── app.go                 # Wails 应用入口
├── internal/
│   ├── crypto/           # AES 加密模块
│   ├── database/         # GORM + SQLite 配置
│   └── models/           # 数据模型
├── frontend/
│   ├── src/
│   │   ├── views/        # 页面组件
│   │   ├── components/   # 通用组件
│   │   └── api/          # Go 绑定调用
│   └── bindings/         # Wails v3 自动生成的绑定
├── wails.json
└── build/                # 打包资源
```

## 核心模块

### 1. 数据库配置 (No CGO)

```go
// 使用纯 Go SQLite 驱动
import "gorm.io/driver/sqlite"

// 初始化
db, _ := gorm.Open(sqlite.Open("data.db"), &gorm.Config{})
```

### 2. AES 加密

```go
// AES-256-GCM 加密
func Encrypt(plaintext, key []byte) ([]byte, error)
func Decrypt(ciphertext, key []byte) ([]byte, error)
```

- 密钥派生：PBKDF2 (主密码 → 32字节密钥)
- 模式：GCM (认证加密)

### 3. 数据模型

```go
type PasswordEntry struct {
    gorm.Model
    Title       string
    Username    string
    Password    string  // AES 加密存储
    URL         string
    Category    string
    Notes       string
}
```

### 4. 前端 API 调用

```typescript
// Wails v3 绑定调用
import { CreateEntry, GetEntries, DecryptPassword } from '../bindings/else-toolbox/app'

// 使用示例
const entries = await GetEntries()
await CreateEntry({ title, username, password, url })
```

## 快速开始

```bash
# 安装前端依赖
cd frontend && bun install

# 开发模式
task dev
# 或
wails3 dev

# 构建
task build
```

## 安全设计

1. **主密码**：首次启动设置，使用 PBKDF2 派生密钥
2. **加密存储**：所有密码字段 AES-256-GCM 加密
3. **无网络**：纯本地存储，无云端同步
