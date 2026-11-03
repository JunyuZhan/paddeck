# PadDeck

> 把闲置 iPad 变成 Homelab 监控大屏。Agentless、单二进制、Safari 10 可用。

PadDeck is a self-hosted homelab dashboard designed for **old iPads stuck on iOS 10 / Safari 10**.
It collects Linux / macOS metrics, Docker containers, Proxmox VE guests, iKuai router stats and
HTTP probe results — no agent required — and renders them with a tiny ES5 frontend built for
always-on, wall-mounted displays.

PadDeck 是一个自托管的局域网监控面板，专为 **iOS 10 / Safari 10 的老 iPad** 设计。
不要求被监控主机安装任何 agent（能用 SSH 就用 SSH，爱快走设备自带 API），
前端是零构建的 ES5 单页应用，适合挂在墙上长期显示。

**文档**：[设计](docs/design.md) · [开发指南](docs/development.md) · [开发计划](docs/roadmap.md)

## 特性

- **开箱即用**：`paddeck -init` 生成配置、`paddeck -check` 逐项体检、配置自动发现、SSH 密钥自动找
- **老设备友好**：ES5 + XMLHttpRequest + SVG，无框架、无构建、无 CSS 变量/Grid
- **Agentless**：Linux / macOS 走 SSH，PVE 走 `pvesh`，爱快走 Web API
- **6 页 NOC 面板**：概览只报状态（KPI + 告警），主机/容器/虚拟化/服务/趋势各自独立，指标不跨页重复
- **容器归属**：每个容器格标出所在机器（主机色标签 + 左边条），停机容器斜线纹理一眼可辨
- **5 套配色 × 5 种背景**：Mocha / Rosé Pine / Gruvbox / Nord / Latte，极光 / 弥散 / 点阵 / 透视网格 / 纯色
- **历史曲线**：SQLite 每分钟聚合，保留 30 天；主机卡内嵌 6 分钟 CPU sparkline
- **动效克制**：只用合成层位移与静态渐变，rAF 实测帧率低于 24fps 时自动降级
- **单二进制**：Go + 内嵌前端，Docker 镜像约 25MB

## 快速开始

只需要 Go 1.24+，不需要 Node、不需要数据库服务：

```bash
go build -o paddeck ./cmd/paddeck

./paddeck -init          # 生成最小可用 config.yaml
vi config.yaml           # 填上你的机器 / 路由器 / PVE / 网址
./paddeck -check         # 体检：SSH、docker、PVE、爱快、HTTP 逐项自测
./paddeck                # 打开 http://localhost:9080
```

Docker 部署：

```bash
cp config.example.yaml config.yaml
vi config.yaml

mkdir -p keys && ssh-keygen -t ed25519 -f keys/paddeck_ro -N ""
ssh-copy-id -i keys/paddeck_ro.pub you@192.168.1.10

docker compose up -d --build
```

打开 `http://<host>:9080`。

### 配置自动发现

不传 `-config` 时按顺序查找：

```
$PADDECK_CONFIG → ./config.yaml → ./config.local.yaml
→ $XDG_CONFIG_HOME/paddeck/config.yaml → /config/config.yaml
```

数据库默认 `/data/paddeck.db`（Docker），否则 `./data/paddeck.db`；可用 `-db` 或 `$PADDECK_DB` 覆盖。

### SSH 免配置

`collector.ssh_key` 留空时依次尝试：`~/.ssh/id_ed25519` → `~/.ssh/id_rsa` → `~/.ssh/id_ecdsa`
→ `ssh-agent`。所以绝大多数人只要 `ssh-copy-id` 过就能跑，不用再改配置。

## 命令

| 命令 | 作用 |
| --- | --- |
| `paddeck` | 启动面板 |
| `paddeck -init` | 生成最小可用配置 |
| `paddeck -check` | 校验配置并逐项探测所有目标，退出码非 0 表示有目标不通 |
| `paddeck -version` | 版本号 |
| `paddeck -config <path>` / `-db <path>` | 手动指定配置 / 数据库 |

`-check` 输出示例：

```
hosts
  ok    web-01      192.168.1.10    linux Linux 6.12.0 · docker ok (9 containers)
  FAIL  nas         192.168.1.5     dial tcp 192.168.1.5:22: i/o timeout
        hint: ssh youruser@192.168.1.5 — check address, user and that your key is in ~/.ssh/authorized_keys
```

## 配置

完整示例见 [`config.example.yaml`](config.example.yaml)，最小配置：

```yaml
hosts:
  - id: web-01
    address: 192.168.1.10
    user: youruser
    docker: true

services:
  - name: NAS
    url: http://192.168.1.5/
```

| 字段 | 默认值 | 说明 |
| --- | --- | --- |
| `server.listen` | `:9080` | 监听地址 |
| `collector.interval` | `10s` | 主机采集间隔 |
| `collector.docker_interval` | `30s` | Docker 采集间隔 |
| `collector.history_interval` | `60s` | 历史写入间隔 |
| `collector.http_timeout` | `3s` | 探活超时 |
| `collector.ssh_timeout` | `8s` | SSH 超时 |
| `collector.retention_days` | `30` | 历史保留天数 |
| `collector.ssh_key` | 空（自动） | 留空则依次尝试 `~/.ssh/id_ed25519`、`id_rsa`、`id_ecdsa`、ssh-agent |

主机字段：

| 字段 | 说明 |
| --- | --- |
| `id` / `name` | 唯一标识 / 显示名称 |
| `type` | `linux`（默认）/ `macos` / `ikuai` |
| `address` / `user` / `port` | SSH 连接信息（`ikuai` 不用） |
| `api_url` / `username` / `password` | 仅 `ikuai`：默认 `https://<address>:444` |
| `docker` / `docker_sudo` / `docker_path` | 容器采集（群晖路径为 `/usr/local/bin/docker`） |
| `disk_mounts` | 磁盘统计挂载点，默认 `/` |

PVE 节点字段与主机相同，可以列多台；面板节点名取 `name`。

配置里的任意值都支持环境变量展开，密码不必写进文件：

```yaml
password: ${IKUAI_PASSWORD}
```

## 外观

打开页面先出现外观选择页：配色 5 套、背景 5 种，选择记在浏览器 `localStorage`。
也可以直接用 URL 跳过（适合书签/自动化）：

```
http://<host>:9080/?theme=gruvbox&bg=aurora
```

配色 id：`mocha` `rosepine` `gruvbox` `nord` `latte`；背景 id：`aurora` `mesh` `dots` `retro` `none`。

## iPad Kiosk 设置（不越狱）

1. Safari 打开面板地址（可带 `?theme=…&bg=…`），分享 → **添加到主屏幕**
2. 设置 → 显示与亮度 → 自动锁定 → **永不**
3. 设置 → 辅助功能 → 引导式访问 → 开启；进入后三击侧边键锁定
4. 插电，亮度 40–50%

> iPad 4 最高只能到 iOS 10.3.3。Grafana / Uptime Kuma / Home Assistant 等现代前端在
> Safari 10 上会白屏，这正是 PadDeck 存在的原因。
>
> 已知限制：iOS 无法隐藏顶部状态栏（时间/WiFi/电池），只能让它与顶栏同色。

## 架构

```
iPad Safari ──5s 轮询──▶ PadDeck (Go)
                           ├── SSH ──▶ Linux /proc, df, /proc/net/dev
                           ├── SSH ──▶ macOS top / vm_stat / sysctl / df
                           ├── SSH ──▶ docker ps / stats
                           ├── SSH ──▶ pvesh（节点 + qemu + lxc）
                           ├── HTTPS ─▶ 爱快 Web API（登录 + sysstat）
                           ├── HTTP ──▶ 服务探活
                           └── SQLite 历史（1 分钟粒度，30 天）
```

## API

| 路径 | 说明 |
| --- | --- |
| `GET /api/snapshot` | 全量快照 |
| `GET /api/history?host=<id>&range=6h` | 主机历史（1h/6h/24h/7d） |
| `GET /api/health` | 健康检查 |

## 安全

- 面板**没有登录**，只应部署在可信内网。
- SSH 密钥建议单独生成，只授予只读权限。
- 不要在配置里写明文密码，用 `${ENV_VAR}`。
- `config.yaml`、`keys/`、`data/` 已在 `.gitignore`，请不要提交。

## 参与

见 [CONTRIBUTING.md](CONTRIBUTING.md)。安全问题请按 [SECURITY.md](SECURITY.md) 私下报告。

## License

MIT © JunyuZhan

## 文档索引

| 文档 | 内容 |
| --- | --- |
| [docs/design.md](docs/design.md) | 架构、采集协议、数据模型、前端设计取舍 |
| [docs/development.md](docs/development.md) | 本地开发、部署、加采集器、加主题/背景、排障 |
| [docs/roadmap.md](docs/roadmap.md) | 已完成 / 下一步 / 视觉风格评估 / 技术债 / ADR |
| [CHANGELOG.md](CHANGELOG.md) | 版本变更记录 |
