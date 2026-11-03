# PadDeck 开发指南

面向要改代码/加采集器/加主题的人。设计取舍见 [design.md](design.md)，下一步计划见 [roadmap.md](roadmap.md)。

## 1. 环境要求

| 依赖 | 版本 | 用途 |
| --- | --- | --- |
| Go | 1.24+ | 编译后端 |
| Python | 3.9+ | 只用来跑 `tools/themes.py` 生成主题（不参与构建） |
| Docker + docker-compose | 任意近期版本 | 部署 |
| Git | — | — |

不需要 Node/npm，前端**没有构建步骤**。

国内网络建议先设：

```bash
export GOPROXY=https://goproxy.cn,direct
```

## 2. 仓库结构

```
cmd/paddeck/main.go        入口：加载配置 → 开库 → 起采集循环 → 起 HTTP
internal/config/           YAML 配置解析、默认值、校验
internal/collector/
  manager.go               采集调度、快照组装、告警规则、CPU 历史环形缓冲
  ssh.go                   SSH 连接封装（只读密钥）
  host.go                  Linux 采集器（/proc、df、/sys）+ 网卡速率
  mac.go                   macOS 采集器（top / vm_stat / sysctl / df）
  ikuai.go                 爱快路由 Web API 采集器
  docker.go                Docker 容器列表与 stats
  pve.go                   Proxmox 节点与客户机
  probe.go                 HTTP 探活（状态码 / 延迟 / TLS 天数）
internal/store/store.go    SQLite 历史（1 分钟聚合、30 天、查询降采样）
internal/api/api.go        /api/snapshot /api/history /api/health + 静态文件
web/                       前端（go:embed 打进二进制）
  index.html               6 页骨架 + 选择页 + 启动页
  app.js                   渲染、导航、手势、主题/背景切换、自适应降级
  style.css                母版样式（Mocha 调色板 + 全部布局）
  theme-*.css              生成物：5 套完整主题
  themes.js                生成物：主题元数据（前端取色 / 预览卡）
tools/themes.py            主题生成器
docs/                      设计文档、开发指南、路线图
deploy/                    备用部署样例
```

## 3. 本地开发

```bash
# 1. 配置（gitignore，含真实地址）
cp config.example.yaml config.local.yaml
vi config.local.yaml          # listen: ":9088" 避免和线上的 9080 冲突

# 2. 准备只读密钥（没有也行，SSH 层会自动找 ~/.ssh 下的 key 或 ssh-agent）
ssh-keygen -t ed25519 -f ./keys/paddeck_ro -N ""
ssh-copy-id -i ./keys/paddeck_ro.pub you@your-host

# 3. 体检（改了配置之后先跑这个，比在面板里看 OFFLINE 快）
GOPROXY=https://goproxy.cn,direct go run ./cmd/paddeck -check -config config.local.yaml

# 4. 跑
GOPROXY=https://goproxy.cn,direct go run ./cmd/paddeck \
  -config config.local.yaml -db ./data/dev.db
```

打开 `http://127.0.0.1:9088`。

### 开箱即用的三件事（改配置/交付前必看）

| 能力 | 实现 | 用户得到什么 |
| --- | --- | --- |
| 配置发现 | `config.Resolve()`：`$PADDECK_CONFIG` → `./config.yaml` → `./config.local.yaml` → `$XDG_CONFIG_HOME/paddeck/config.yaml` → `/config/config.yaml` | 不用记 `-config` 参数 |
| 配置体检 | `paddeck -check`：逐台 SSH / docker / pvesh / 爱快登录 / HTTP 探活，失败给修复提示 | 装完立刻知道哪里配错 |
| 密钥自动查找 | `SSH.authMethods()`：`ssh_key` 留空时依次试 `~/.ssh/id_ed25519`、`id_rsa`、`id_ecdsa`、`ssh-agent` | `ssh-copy-id` 过就能跑 |

另外配置支持 `${VAR}` / `${VAR:-默认值}` 展开，把密码放环境变量里，配置文件可以安全地进仓库模板：

```yaml
password: ${IKUAI_PASSWORD}
```

`paddeck -init` 会生成一份带注释的最小配置；`-force` 覆盖已存在的文件。

**改前端**：直接改 `web/` 下的文件，然后**重启进程**（文件是 `go:embed` 编进二进制的，
不重启不会生效）。改完记得重跑 `python3 tools/themes.py`（如果你动了 `style.css` 里的颜色）。

**调试页面**：支持 hash 直达 —— `#overview` `#hosts` `#containers` `#pve` `#services` `#trend`。

**跳过外观选择页**：`?theme=gruvbox&bg=aurora`（配色 `mocha/rosepine/gruvbox/nord/latte`，
背景 `aurora/mesh/dots/retro/none`）。

**强制降级档**：`http://…/?lowfx`，用来对比"是不是动效在拖慢 iPad"。

## 4. 构建与部署

生产环境是一台 Docker 主机（本文档里叫 `mu`，`/opt/paddeck`）。

```bash
# 交叉编译
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 GOPROXY=https://goproxy.cn,direct \
  go build -trimpath -ldflags="-s -w" -o paddeck-linux ./cmd/paddeck

# 推送 + 重建容器（Dockerfile 只是把二进制塞进 alpine，秒级）
scp -q paddeck-linux mu:/opt/paddeck/paddeck
ssh mu 'cd /opt/paddeck && docker-compose up -d --build'
```

`/opt/paddeck` 只需要三个东西：

```
paddeck            # 二进制
config.yaml        # 真实配置（chmod 600，不进 git）
docker-compose.yml # 端口 9080，挂载 config/keys/data
```

也可以直接在服务器上 `docker compose up -d --build` 用 `Dockerfile` 里的多阶段构建，
不需要本地交叉编译。

部署后自检：

```bash
curl -s http://<host>:9080/api/health
curl -s http://<host>:9080/api/snapshot | python3 -m json.tool | head
```

## 5. 测试与 CI

```bash
go vet ./...
go test ./...
```

CI（`.github/workflows/ci.yml`）在每次 push/PR 跑 vet + test + 构建。

当前的测试覆盖：

| 文件 | 覆盖 |
| --- | --- |
| `internal/collector/host_test.go` | `/proc/stat` 解析、CPU 差值、df、内存 |
| `internal/collector/docker_test.go` | `docker ps/stats` JSON 解析 |
| `internal/collector/ikuai_test.go` | iKuai 登录/`sysstat`/`iface_check` 响应解析 |
| `internal/config/config_test.go` | 默认值、校验、错误分支 |
| `internal/store/store_test.go` | 写入、查询、降采样 |

加解析类的逻辑时**优先补 fixture 测试**——采集器的坑几乎都在解析上。

## 6. 后端：加一种设备

采集器的契约只有一个方法：

```go
// internal/collector/manager.go
type metricsCollector interface {
    Collect(ctx context.Context, key string) HostMetrics
}
```

`HostMetrics`（`internal/collector/host.go`）是统一出参，字段按需填：

| 字段 | 说明 |
| --- | --- |
| `CPU` | 百分比 0–100 |
| `MemUsed` / `MemTotal` | 字节 |
| `DiskUsed` / `DiskTotal` | 字节（取 `disk_mounts` 指定的挂载点） |
| `Load1/5/15` | 负载 |
| `Uptime` | 秒 |
| `Temp` | 摄氏度（没有就留 0） |
| `Kernel` / `Hostname` | 展示用 |
| `NetRx` / `NetTx` | 字节/秒（由两次采样差值算出） |
| `Clients` / `Conns` / `WANUp` / `WANDown` / `WANIP` / `WANStatus` | 路由类设备专用 |
| `Online` / `LastSeen` / `Err` | 状态与错误 |

**新增一种设备的步骤**：

1. 在 `internal/collector/` 新建文件，实现 `Collect`；
2. 在 `internal/config/config.go` 的 `Validate()` 里把新的 `type` 加进白名单
   （`linux` / `macos` / `ikuai` 已有，`ikuai` 需要 `username`+`password`）；
3. 在 `internal/collector/manager.go` 的 `NewManager()` 里按 `h.Type` 挑采集器；
4. 前端加图标与配色映射（`hostIcon()` / `hostAccent()` / `ICONS`）；
5. 补一个解析测试。

**告警规则**在 `manager.go` 的 `buildAlerts()`：主机离线、CPU ≥ 90、内存 ≥ 90%、磁盘 ≥ 90%、
服务探活失败。四个阈值是写死的常量，改起来只动这一处。

**历史采样**：`writeHistory()` 每 `history_interval`（默认 60s）写一行；`store.History()`
查询时自动降采样到 ≤ 240 点。启动后第一分钟不写（避免脏数据）。

## 7. 后端：API

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/snapshot` | 最新快照（主机/容器/PVE/服务/告警） |
| GET | `/api/history?host=<id>&range=6h` | 历史采样，`range` 支持 1h/6h/24h/7d |
| GET | `/api/health` | `{"ok":true}` |
| GET | `/` | 静态前端 |

快照结构见 [design.md §5](design.md)。前端只依赖这个结构，改字段要同步改 `web/app.js`。

## 8. 前端开发

### 硬约束（Safari 10 / iPad 4）

| 禁止 | 用这个 |
| --- | --- |
| `let/const`、箭头函数、模板字符串、解构、`async/await` | `var` + `function` |
| `fetch` | `XMLHttpRequest` |
| CSS 变量、CSS Grid、`gap` | 内联值、flex + 百分比宽高 |
| `env(safe-area-inset-*)` | `navigator.standalone` 判断 + 固定 20px |
| `Element.closest`（可用但避免）、`IntersectionObserver` | 手写 `while` 遍历 `parentNode` |
| `filter: blur()`、`backdrop-filter`、大尺寸 `box-shadow` | 预渲染渐变 / 静态阴影 |
| 全屏 canvas 60fps 动画 | 合成层 `translate3d` / 静态渐变 |

### 结构

`web/app.js` 分成这些块（顶部注释有分隔）：

1. **palette** — `THEMES`（来自 `themes.js`）、`C`（当前主题的色值）、`ACC`（6 个强调色）
2. **icons** — 内联 SVG 路径表 + 关键词匹配规则
3. **helpers** — 单位格式化、XHR、`refs()` 元素引用缓存
4. **view()** — 原地更新机制（见下）
5. **各页 build/update** — `buildXxx()` 生成 HTML 字符串，`updateXxx()` 只改文本/属性
6. **trend** — 手写 SVG 折线
7. **effects** — 聚光、倾斜
8. **navigation** — 翻页动画、跟手拖动
9. **picker** — 主题/背景选择页
10. **refresh / boot / init**

### 原地更新（重要）

```js
view(containerId, items, structureKey, buildFn, updateFn);
```

- `structureKey` 变了（条目增减、类型变化）→ 整块重建；
- 没变 → 逐个调用 `updateFn(tile, item)`，只改 `textContent` / `style.width` / `className`。

不要在 `updateFn` 里重建 DOM，否则流光、倾斜、sparkline 每 5 秒被打断一次。
用 `refs(el)` 缓存查询结果，别每帧 `querySelector`。

### 加一页

1. `web/index.html` 加 `<section class="page" id="page-xxx">`；
2. `app.js` 的 `PAGES` 数组加一项（`id` / `label` / `acc` 强调色）；
3. 写 `renderXxx()`，在 `render()` 里调用；
4. 需要翻页过渡的话，样式已经由 `.page` 那套类处理，不用额外写。

### 动效预算

能常驻的只有：合成层位移（`translate3d`）、静态渐变、静态阴影。任何会触发整屏重绘的
（改 `background-position`、`filter`、大范围 `box-shadow` 动画）都放 `.lowfx` 白名单里。

## 9. 主题与背景

### 主题（配色）

```bash
# 1. 在 tools/themes.py 里加一个 theme("id", "名字", "描述", {调色板})
# 2. 重新生成
python3 tools/themes.py
# 3. 重启进程（CSS 是 embed 进去的）
```

`tools/themes.py` 的工作方式：把 `web/style.css` 里属于 Mocha 调色板的字面值替换成
占位符，再按目标调色板填回去，输出 `web/theme-<id>.css`；同时生成 `web/themes.js`
给前端取色。**所以 `style.css` 必须始终保持 Mocha 的色值**，不要在里面直接写别的主题色。

调色板需要的 token 见 `docs/design.md §6`，最少要定义：
`bg / panel / border / hair / offbg / track / dotoff / text / name / num / sub1 / sub0 /
ov1 / ov0 / ov2 / a1..a6 / ok / warn / crit`，其余（警告底、3D 阴影）会自动推导。

### 背景

背景是 `#bgfx` 上的一个 class：`fx-none` / `fx-aurora` / `fx-mesh` / `fx-dots` / `fx-retro`。
加一种只要在 `style.css` 写一个 `.fx-xxx { background-image: … }`，然后在 `app.js` 的
`BGS` 数组登记，选择页会自动出现预览卡。

**背景只能用径向/线性渐变或静态图案**。要用 canvas 或 `filter` 的方案请先读
[roadmap.md](roadmap.md) 里的性能评估。

## 10. 排障

| 现象 | 先查 |
| --- | --- |
| 不知道哪里配错了 | 先跑 `paddeck -check`，它会逐项给出失败原因和修复提示 |
| iPad 改了没生效 | Safari 缓存。强制下拉刷新；仍不行就在 URL 后加 `?v=2` 再打开 |
| 页面卡顿/掉帧 | 开 `?lowfx` 对比。若明显变快说明是动效预算问题，把新加的效果挪进 `.lowfx` |
| 连不上页面 | `curl /api/health`；容器 `docker logs paddeck` |
| 某台主机离线 | `ssh -i keys/paddeck_ro <user>@<host> 'uname -a'` |
| 爱快离线 | curl 一下 `https://<ip>:444/Action/login`；确认账号密码、`api_url` 端口 |
| Mac 离线 | 系统设置里「远程登录」是否开着；`~/.ssh/authorized_keys` 是否含只读公钥 |
| PVE 节点名重复 | 两台独立 PVE 都叫 `pve`，用配置里的 `name` 覆盖（已实现） |
| 容量/温度显示 `--` | 该设备没上报这个指标，属于正常 |

## 11. 提交规范

- 提交信息用 `type(scope): subject`，type 取 `feat` / `fix` / `docs` / `style` / `refactor` / `test`
- 改了 `web/style.css` 的颜色 → **必须**同步生成主题文件并一起提交
- 改了采集输出 → 同步更新 `docs/design.md §3` 与 `HostMetrics` 字段表
- 配置项有增删 → 同步 `config.example.yaml`、README 配置表、`internal/config` 默认值
