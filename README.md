# 施工扬尘与环境指标监测处置平台

本项目是面向施工现场的离线可运行环境监管系统。它接入本地模拟设备的 PM2.5、PM10、噪声、温湿度和风速数据，围绕设备治理、质量隔离、版本化规则、告警处置、校准维护、趋势统计、日报导出与审计形成完整闭环。

## 架构

- `cmd/server`：进程入口、信号和优雅停机。
- `internal/domain`：工地权限、设备、遥测、规则、告警、维护、身份、审计和报表实体及状态机。
- `internal/application`：接入、规则、告警、设备、维护、监控、报表、认证和异步任务用例，接口靠近调用方。
- `internal/repository/postgres`：pgx 连接池、版本化迁移和持久化适配器；未配置数据库时使用确定性内存适配器演示。
- `internal/transport/http`：Gin `/api/v1` API、参数校验与稳定错误响应。
- `internal/platform`：时钟、ID、文件、通知、outbox 与事务抽象。
- `web`：Vue 3、TypeScript、Vite、Pinia 和 Element Plus 操作台。
- `api/openapi`：OpenAPI 3.0 JSON 契约；`migrations` 和 `scripts` 提供运维入口。

核心不变量：设备、测点和采样时间构成幂等身份；原始观测不可覆盖，校正总是追加新记录；迟到观测可以留存但不重复生成告警；evaluation 固定规则版本和时区；受控重算追加新结论而不删除旧结论；设备离线与环境超标使用不同告警 kind、merge key 和统计口径。

## 本地运行

要求 Go 1.24+ 和 Node.js 20+。不配置 `DATABASE_URL` 时后端自动启用本地内存适配器，并写入固定演示数据。

```bash
cp .env.example .env
go run ./cmd/server
npm --prefix web install
npm --prefix web run dev
```

后端默认监听 `http://localhost:8080`，前端监听 `http://localhost:5173`。健康检查为 `/healthz`，依赖就绪检查为 `/readyz`，Prometheus 文本指标入口为 `/metrics`。演示账号 `demo.supervisor` 的种子密码为 `DustDemo!2026`，只用于本地确定性演示；登录后使用 `Authorization: Bearer <access_token>` 访问业务接口。开发环境仍允许 `X-Demo-Actor: demo-supervisor` 进行离线联调，生产环境关闭该通道。

## 容器运行与迁移

```bash
docker compose up --build
```

Compose 启动 API 与 PostgreSQL，并把两者放在内部控制网络中。应用容器使用 UID 10001 非 root 用户，Dockerfile 基于官方多架构 Go/Alpine 镜像，可在 amd64 和 arm64 上构建。启动时自动按名称顺序执行嵌入迁移；运维可按文件名顺序执行 `migrations` 下的入口文件。确定性演示数据由应用启动过程通过领域仓储幂等写入，也可用脚本启动同一流程：

```powershell
.\scripts\start-demo.ps1 -DatabaseUrl $env:DATABASE_URL
```

## API 示例

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"demo.supervisor","password":"DustDemo!2026"}'

curl -X POST http://localhost:8080/api/v1/telemetry/batches \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer <access_token>' \
  -d '{"batch_id":"demo-batch-1","samples":[{"device_code":"DUST-EAST-001","schema_id":"schema-pm10","value":182,"sampled_at":"2026-08-23T08:00:00Z"}]}'
```

列表和时间序列入口使用显式页码/游标、排序和字段白名单。错误响应始终包含 `code`、`message`、`field_errors` 和 `request_id`。完整接口见 `api/openapi/monitoring-api.json`。

## 状态机

- 设备：`registered -> online/offline/maintenance -> replaced/retired`，替换必须关联新设备。
- 告警：`open -> acknowledged -> dispatched/recovering -> recovered -> closed`，恢复后再次触发可重新打开。
- 工单：`assigned -> accepted -> processing -> resolved -> verified`。
- 规则版本：`draft -> active -> superseded -> retired`。

非法跳转返回稳定业务错误。关键写入使用幂等键、数据库唯一/外键/检查约束、必要索引和乐观版本；审计条目记录操作者、来源、前后差异、原因、request_id 与 UTC 时间。

## 测试与已执行验证

```bash
gofmt -l .
go test ./...
go test -race ./...
go vet ./...
npm --prefix web test -- --run
npm --prefix web run typecheck
npm --prefix web run build
```

PostgreSQL 集成测试需要 `TEST_DATABASE_URL`。领域测试覆盖状态机、观测幂等与追加校正、规则版本/时区、迟到数据抑制告警、环境/离线告警分治、HTTP 权限与错误结构；前端测试覆盖两类告警指标分离。

## 配置与限制

配置见 `.env.example`；生产环境必须显式设置至少 32 字符的 `ACCESS_TOKEN_KEY`。所有时间入库存 UTC，界面按工地时区显示。文件上传校验扩展名、声明 MIME、实际内容、大小、摘要和安全路径，下载只能经过工地资源授权接口，响应不会暴露磁盘路径。日志只记录关联字段，不记录密码、令牌、签名正文或附件内容。通知和设备协议使用本地适配器，不依赖 CDN、云地图、在线模型、真实短信或推送平台。

当前演示适配器不会模拟真实硬件网络抖动；生产部署应配置 PostgreSQL，并将通知与本地协议接口替换为现场适配器。访问令牌使用本地 HMAC 签名，仓库只包含开发占位值，不包含任何真实凭据。
