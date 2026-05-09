# SubSage — Claude Code 协作指南

自托管订阅管理工具，**AI Agent Native**：单二进制部署 + REST API 给 Agent 调用。当前 v0.4.3。

本文件只放**协作时容易踩坑、又不在其他文档里**的规则。其余请直接读：

- 架构、技术栈、设计规范、API 列表、数据模型 → [docs/architecture.md](docs/architecture.md)
- 发版完整流程 → [.agents/workflows/release.md](.agents/workflows/release.md)
- 用户视角 → [README.md](README.md) / [README_zh.md](README_zh.md)

---

## 常用命令

```bash
# 后端开发（默认读 ../data/sage.db，端口 8321）
cd backend && go run .

# 前端开发（自动代理 /api → :8321）
cd frontend && npm install && npm run dev

# 前端生产构建（输出到 backend/static/，被 Go embed）
cd frontend && npm run build

# 跨平台二进制
CGO_ENABLED=0 go build -C backend -ldflags="-s -w" -trimpath -o ../dist/subsage .
```

构建无 CGO 依赖（`modernc.org/sqlite` 是纯 Go 实现），任意平台直接交叉编译。

---

## 非显然规则

### 1. 前端构建产物 = `backend/static/`，不要直接改

`frontend/vite.config.js` 把 `build.outDir` 指向 `../backend/static/`，Go 用 `embed` 打包进二进制。`backend/static/` 在 `.gitignore` 里。**永远改 `frontend/src/`，跑 `npm run build` 重新生成。**

### 2. 版本号有唯一来源 + 多处同步

`frontend/src/version.js` 是 single source of truth（被 Sidebar / Settings / Login 引用）。改版本时还要同步：

- `docs/architecture.md` 顶部的「当前版本」行
- `README.md` 和 `README_zh.md` 的版本历史表

具体步骤见 [.agents/workflows/release.md](.agents/workflows/release.md)。

### 3. 新增订阅字段要改 5 个地方

`subscriptions` 表的字段散落在多处，少改一处就 panic 或字段丢失：

1. `backend/database/db.go` — 在 `addCols` 里加幂等 ALTER（不要改 `CREATE TABLE`，老库不会重跑）
2. `backend/model/models.go` — `Subscription` struct
3. `backend/handler/util.go` — `SubColumns` 常量 + `ScanSub` 的 `Scan(...)` 参数顺序必须对齐
4. `backend/handler/sub.go` — `Create` / `Update` / `Patch` 的 SQL 与参数列表
5. `frontend/src/components/EditSubModal.svelte` — 表单 + 提交 payload

PATCH 语义：零值/空字符串 = 「不更新」，详见 `Patch` 的合并逻辑。

### 4. 数据库迁移：幂等 ALTER，不动 `CREATE TABLE`

老库已经存在的表不会重跑 `CREATE TABLE`。新加列走 `addCols` 数组的「试 SELECT → 失败就 ALTER ADD COLUMN」模式（见 `db.go:91-109`）。**不要修改已发布的 CREATE TABLE 语句**——会让新库和老库不一致。

### 5. i18n 双语强约束

零硬编码字符串（v0.2.3 已审计承诺）。任何前端面向用户的字符串：

- 加 key 到 `frontend/src/i18n/en.js` **和** `frontend/src/i18n/zh.js`，缺一个 fallback 会回退到英文
- 模板 `{name}` 占位符通过 `t('key', { name: 'foo' })` 注入
- 不要在 `.svelte` 里写中文/英文字面量

### 6. Agent API 与 Web API 共用 handler

`/api/*`（JWT 认证）和 `/api/agent/*`（Token 认证）在 `main.go` 里挂的是**同一组 handler**。改 handler 行为时记得两条路径都受影响。Token 中间件读 DB，支持热刷新（regenerate token 后无需重启）。

### 7. lazy advance 续费日期

`auto_renew=true` 的订阅在 List/Get 时会自动把过期的 `next_renewal` 推进到下一个未来日期（`sub.go:lazyAdvance`）。这是异步 goroutine 写回 DB——**测试统计准确性时注意**：列表读一次后下次 stats 数字可能变化。

---

## 协作硬约束

- **不要 `git push`**——push 仅用于跨设备同步或发版，等明确指令。
- **不要把发版当作 push 触发**——部署走 [.agents/workflows/release.md](.agents/workflows/release.md)：手动 build 二进制 + `gh release upload` + `docker buildx --push`。CI 里没有自动发布。
- **设计系统已成规范**（[docs/architecture.md](docs/architecture.md) 的色彩/字体/布局段）。如果在写新 UI 时发现某条规范不合适，**先指出再改文档**，不要默默偏离。
- **设计 token 用 CSS 变量**（`frontend/src/styles/global.css` 已定义 `--primary` / `--bg` / `--card` 等），不要写死十六进制。

---

## 目录速查

```
backend/
  main.go            入口、路由挂载
  database/db.go     SQLite 初始化 + 迁移
  handler/           HTTP handler（auth/sub/stats/agent/export/settings）
  middleware/        JWT、Token、Rate limit
  model/models.go    数据模型
frontend/src/
  pages/             Overview / SubList / Calendar / Agent / Settings / Login
  components/        Sidebar / EditSubModal / Charts / Toast / UpcomingList / AgentSidebar
  api/index.js       前端调 /api 的封装
  i18n/{en,zh}.js    所有 UI 文案
  stores/index.js    主题 / 当前页 / 用户偏好
  styles/global.css  CSS 变量 + 全局样式
data/sage.db         SQLite（gitignored）
.agents/workflows/   Agent 工作流定义
```
