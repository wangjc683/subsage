# SubSage — Development Reference

## 概述
SubSage is a self-hosted subscription management tool. **AI Agent Native**: it exposes itself as a tool for your existing AI Agent via REST API.

**当前版本**: v0.2.1

## 产品定位

- **Target Users**: General users with an AI Agent
- **Core Idea**: AI Agent Native — copy one command to connect
- **交互模式**: 双入口
  - 人类入口：Web Dashboard（总览、订阅管理、日历）
  - Agent 入口：REST API（Token 认证）

## 技术栈

| 层 | 技术 |
|---|---|
| 后端 | Go 1.25 + Echo + SQLite (modernc.org/sqlite, 纯Go无CGO) |
| 前端 | Svelte 4 + Vite |
| 认证 | 用户名 + 密码 → bcrypt → JWT（Web）+ API Token（Agent） |
| 数据库 | SQLite 单文件 |
| 图表 | Chart.js |
| Excel 导出 | excelize (Go) |
| 汇率 | open.er-api.com 免费版，每日缓存 |
| 部署 | Docker / 单二进制 + systemd + Caddy 反代 |

## 设计规范

### 色彩
```
主色:      #3D7C5F  (深橄榄绿, 灵感来自小米SU7橄榄绿车漆)
主色亮:    #4E9B78  (hover / accent)
主色暗:    #2C5A44  (pressed / active)
主色淡:    #3D7C5F20  (背景 tint)
主色极淡:  #3D7C5F0A  (选中/聚焦背景)

暗色模式:
  bg:      #141414
  surface: #1A1A1A
  card:    #212121
  hover:   #2A2A2A
  border:  #2A2A2A / #333333

亮色模式:
  bg:      #FAFAFA
  surface: #FFFFFF
  card:    #F5F5F5
  hover:   #EEEEEE
  border:  #E6E6E6

文字:
  primary:   #141414 (亮) / #E4E4E4 (暗)
  secondary: #777777 (亮) / #999999 (暗)
  tertiary:  #929292 (亮) / #666666 (暗)

语义色:
  error:   #ED3F3F
  success: #44B931
  warning: #FFB020
```

### 字体
- 显示字体（标题/数字）: DM Sans — 几何感强，干净
- 正文字体: Inter — 高可读性
- CSS font-family: `'DM Sans', 'Inter', -apple-system, sans-serif`

### 风格参考
- AFFiNE 设计语言：干净克制、工具感、专业
- 圆角统一 8-12px
- 层级靠背景色微妙变化（alpha 透明度）
- 保持呼吸感，信息密度适中

### 布局
- 侧边栏 (60-80px) + 主区域，自适应
- 桌面：左侧固定侧边栏
- 手机：侧边栏折叠为汉堡菜单或底部导航

## 页面结构

### 导航
总览 → 订阅 → 日历 → Agent → 设置

### 总览页
- 4-stat 栏：月均支出 / 年均支出 / 活跃订阅 / 日均成本
- Agent 状态 Banner（动态：就绪/活跃 + 一键复制配置）
- 即将到期列表 + 月度趋势图（并列）
- 分类支出横条图
- 按币种（多币种时显示）

### 订阅管理页
- 智能分类筛选 pill（仅显示有订阅的分类 + 更多按钮）
- 支持自定义分类（📌 默认图标）
- 新增/编辑弹窗（分 4 个区域：基本信息、费用、时间与状态、其他）

### Agent 页
- API 配置（连接地址 + 连接密钥 + 重新生成）
- 一键复制接入文本（自然语言，包含真实 Token）
- 最近 Agent 调用活动（人类可读描述）
- API 端点参考（默认折叠）

### 日历页
- 日历视图展示续费日期

### 设置页
- 基准货币、汇率、数据导入导出

## 架构

### 目录结构
```
sage/
├── CONTRIBUTING.md      # This file - development reference
├── backend/
│   ├── main.go          # 入口
│   ├── handler/         # HTTP handlers
│   │   ├── auth.go      # 认证
│   │   ├── sub.go       # 订阅 CRUD
│   │   ├── stats.go     # 统计
│   │   ├── agent.go     # Agent 状态
│   │   ├── export.go    # 导入导出
│   │   ├── settings.go  # 设置
│   │   └── util.go      # 工具函数
│   ├── middleware/       # JWT + Token 认证 + Agent 日志
│   ├── database/        # SQLite 初始化 & 迁移
│   └── model/           # 数据模型
├── frontend/
│   ├── src/
│   │   ├── App.svelte
│   │   ├── pages/
│   │   │   ├── Overview.svelte   # 总览
│   │   │   ├── SubList.svelte    # 订阅管理
│   │   │   ├── Calendar.svelte   # 日历
│   │   │   ├── Agent.svelte      # Agent 接入
│   │   │   ├── Settings.svelte   # 设置
│   │   │   └── Login.svelte      # 登录
│   │   ├── components/
│   │   ├── stores/
│   │   ├── api/
│   │   └── styles/
│   └── public/
└── data/
    └── sage.db
```

### 数据模型

**subscriptions 表:**
```sql
CREATE TABLE subscriptions (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    category TEXT NOT NULL,        -- ai/video/music/software/dev/cloud/security/app/gaming/membership + 自定义
    status TEXT DEFAULT 'active',
    price REAL NOT NULL,
    original_price REAL,
    discount_note TEXT DEFAULT '',
    currency TEXT DEFAULT 'USD',
    cycle TEXT DEFAULT 'monthly',   -- monthly/yearly/quarterly/weekly/lifetime
    payment_method TEXT,
    start_date TEXT,
    next_renewal TEXT,
    url TEXT,
    notes TEXT,
    remind_days INTEGER DEFAULT 3,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now'))
);
```

**agent_logs 表:**
```sql
CREATE TABLE agent_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    method TEXT NOT NULL,
    path TEXT NOT NULL,
    user_agent TEXT DEFAULT '',
    created_at TEXT DEFAULT (datetime('now'))
);
-- 自动清理：保留最近 200 条
```

### API 设计

**认证:**
- `POST /api/auth/setup` — 初始化
- `POST /api/auth/login` — 登录 → JWT
- `GET  /api/auth/status` — 检查是否已初始化

**订阅 CRUD (JWT 认证):**
- `GET /api/subs` · `POST /api/subs` · `GET /api/subs/:id` · `PUT /api/subs/:id` · `DELETE /api/subs/:id`

**统计:**
- `GET /api/stats/overview` · `GET /api/stats/by-category` · `GET /api/stats/monthly-trend`

**Agent (JWT 认证):**
- `GET /api/agent/status` — Agent 活动状态 + 调用日志

**Agent API (Token 认证, 无需登录):**
- 所有 `/api/agent/*` 路由使用 `X-API-Token` header 认证
- `GET /api/agent/subs` · `POST /api/agent/subs` · `PUT /api/agent/subs/:id` · `DELETE /api/agent/subs/:id`
- `GET /api/agent/stats/overview` · `GET /api/agent/stats/by-category` · `GET /api/agent/stats/upcoming`

## 分类预设

| ID | 名称 | 图标 |
|---|---|---|
| ai | AI 服务 | 🤖 |
| video | 视频 | 🎬 |
| music | 音乐 | 🎵 |
| software | 软件工具 | 💻 |
| dev | 开发者 | 🔧 |
| cloud | 云服务 | ☁️ |
| security | 安全隐私 | 🛡️ |
| app | 应用 | 📱 |
| gaming | 游戏 | 🎮 |
| membership | 会员 | 👑 |

支持用户自定义分类（默认图标 📌）

## 计费周期

`monthly` · `yearly` · `quarterly` · `weekly` · `lifetime`

## 部署配置

- **Backend Port**: `:8321`
- **Database**: `/data/sage.db` (Docker volume)

## 开发进度

- [x] 后端：项目初始化 + Go modules
- [x] 后端：数据库初始化 + 模型（含 agent_logs 表）
- [x] 后端：认证 API + JWT 中间件
- [x] 后端：订阅 CRUD API
- [x] 后端：统计 API
- [x] 后端：汇率服务
- [x] 后端：导入导出（Excel + JSON）
- [x] 后端：API Token 认证 + Agent API 路由
- [x] 后端：Agent 活动日志 + 状态 API
- [x] 前端：Svelte 项目初始化
- [x] 前端：全局样式 + 主题系统
- [x] 前端：登录页
- [x] 前端：侧边栏 + 路由
- [x] 前端：总览页（4-stat + Agent Banner + 即将到期 + 趋势图 + 分类横条图）
- [x] 前端：订阅管理页（智能筛选 + 自定义分类 + 紧凑表单）
- [x] 前端：日历视图
- [x] 前端：Agent 页面（配置复制 + 活动日志 + 端点参考）
- [x] 前端：设置页
- [x] 前端：PWA
- [x] 部署：Docker + docker-compose
- [ ] 优化：Agent 页面精简（自然语言接入文本 + 人类可读活动日志）
