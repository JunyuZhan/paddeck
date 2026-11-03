# PadDeck 设计文档

> 把闲置 iPad（iOS 10 / Safari 10）改造成局域网 Homelab 监控面板。
> Agentless 采集，Go 单二进制，ES5 前端，MIT 开源。

## 1. 目标与非目标

### 目标
- 在一台 iOS 10.3.3 的 iPad 4 上 7×24 显示局域网设备：Linux/macOS 主机、Docker 容器、
  PVE 虚拟机、HTTP 服务、爱快路由。
- 后端零侵入：不要求被监控主机安装任何 agent；能用 SSH 就用 SSH，不能（爱快）就走设备自带 API。
- 前端在 Safari 10 上稳定运行：ES5、XHR、SVG、flex 布局；所有动效只用合成层位移与静态渐变。
- 单机部署：一个 Docker 容器 + 一个 YAML 配置。
- 可开源：配置与密钥外置，仓库不含真实地址。

### 非目标
- 不做告警推送（仅面板高亮）。
- 不做日志聚合（Dozzle 的职责）。
- 不做多用户/权限系统（内网只读面板）。
- 不支持 Safari 10 以外的旧浏览器兼容性承诺。

## 2. 架构

```
iPad Safari ──HTTP/JSON 轮询 5s──▶ PadDeck (Go, :9080)
                                    │
        ┌───────────────┬───────────┼────────────┬──────────────┐
        ▼               ▼           ▼            ▼              ▼
   Host Collector  Docker      PVE         HTTP Prober     iKuai API
   (SSH)           (SSH)       (SSH)       (HTTP)          (HTTPS)
        │               │           │            │              │
        ▼               ▼           ▼            ▼              ▼
  /proc, df,        ps/stats    pvesh 节点   状态码/延迟/    登录 + sysstat
  /proc/net/dev                 qemu/lxc     证书天数        WAN/终端数
        │               │           │            │              │
        └───────────────┴───────────┴────────────┴──────────────┘
                                    │
                                    ▼
                        内存快照（原子替换）
                                    │
                                    ▼
                        SQLite 历史（1min 聚合 / 30 天）
```

### 组件
| 组件 | 职责 |
|---|---|
| `internal/config` | 加载并校验 YAML |
| `internal/collector/ssh` | SSH 连接池与命令执行 |
| `internal/collector/host` | Linux 指标采集与解析（含网卡速率） |
| `internal/collector/mac` | macOS 指标采集与解析 |
| `internal/collector/ikuai` | 爱快路由 Web API 采集 |
| `internal/collector/docker` | 容器列表与资源采集 |
| `internal/collector/pve` | PVE 节点/VM/CT 采集 |
| `internal/collector/probe` | HTTP 探活 |
| `internal/store` | 内存快照 + SQLite 历史 |
| `internal/api` | 只读 JSON API |
| `web` | ES5 单页应用（内嵌） |
| `tools/themes.py` | 从母版样式渲染出多套主题 CSS |

## 3. 采集

### 3.1 Linux 主机指标
每个采集周期对每台主机执行一条 SSH 命令，输出分节文本（`@@SECTION` 分隔）：

```
@@STAT      /proc/stat 的 cpu 行，读两次（间隔 0.3s）
@@MEM       /proc/meminfo 的 MemTotal/MemAvailable/MemFree/Buffers/Cached
@@LOAD      /proc/loadavg
@@UPTIME    /proc/uptime
@@DISK      df -P（排除 tmpfs/devtmpfs/squashfs）
@@HOST      hostname + uname -r
@@TEMP      /sys/class/thermal/thermal_zone*/temp（取最大）
@@NET       /proc/net/dev
```

解析规则：
- **CPU**：两次 `/proc/stat` 采样的差值（`idle+iowait` 视为空闲），进程内保存上一次采样。
- **磁盘**：默认取 `/`，可通过 `disk_mounts` 追加；每行 `blocks*1024` 换算成字节。
- **温度**：内核给的是毫摄氏度，>1000 时除以 1000；没有传感器则留 0（前端显示 `--`）。
- **网卡速率**：`/proc/net/dev` 所有非 `lo` 接口的 rx/tx 字节累加，与上次采样做差后除以时间差。

### 3.2 Docker
```
docker ps -a --format '{{json .}}'
docker stats --no-stream --format '{{json .}}'
```
- 容器列表每 10s 采集。
- `docker stats` 每 30s 采集（开销较大）。
- 需要 sudo 时使用 `sudo -n docker`。
- 群晖使用 `/usr/local/bin/docker`。

### 3.3 PVE
```
pvesh get /nodes --output-format json
pvesh get /nodes/<node>/status --output-format json
pvesh get /nodes/<node>/qemu --output-format json
pvesh get /nodes/<node>/lxc --output-format json
```

### 3.4 爱快路由（iKuai）
爱快没有可用的 SSH，走它自己的 Web API（自签 HTTPS，默认 444 端口）：

```
POST /Action/login   {"username","passwd":"<md5(password)>","pass":""}   → 下发 sess_key cookie
POST /Action/call    {"func_name":"homepage","action":"show","param":{"TYPE":"sysstat"}}
POST /Action/call    {"func_name":"monitor_iface","action":"show","param":{"TYPE":"iface_check"}}
```

- `sysstat`：每核 CPU 百分比数组、内存总量/可用（KB）、在线终端数、连接数、WAN 上下行速率、uptime、固件版本。
- `iface_check`：WAN 线路状态与 WAN IP，60s 缓存一次。
- 会话失效（返回非 0 code）时丢弃 cookie 重登一次；登录失败则整台主机标记离线。
- 路由没有磁盘与负载指标，前端对应位置改显示在线终端 / 连接数 / WAN 速率。

### 3.5 macOS
macOS 没有 `/proc`，用 `top -l 2`（CPU，约 2s）、`vm_stat` + `hw.memsize`（内存）、
`kern.boottime`（运行时长）、`vm.loadavg`（负载）、`df -k /`（磁盘）采集；网卡速率暂不采集。

### 3.6 HTTP 探活
配置中的每个 URL 执行 GET，3s 超时：
- 记录状态码、耗时、TLS 证书剩余天数。
- 2xx/3xx 视为健康，4xx/5xx 视为异常，超时视为不可达。

## 4. 数据模型

### SQLite
```sql
CREATE TABLE host_samples (
  host_id   TEXT    NOT NULL,
  ts        INTEGER NOT NULL,   -- unix 秒，分钟对齐
  cpu       REAL,
  mem_used  INTEGER,
  mem_total INTEGER,
  disk_used INTEGER,
  disk_total INTEGER,
  load1     REAL,
  PRIMARY KEY (host_id, ts)
);
CREATE INDEX idx_host_ts ON host_samples(host_id, ts);
```
- 每 60s 写入一次。
- 保留 30 天，每小时清理。
- 查询范围：1h / 6h / 24h / 7d，自动降采样到 ≤ 240 个点。

### 内存快照
最新一次完整采集结果，原子替换，API 直接读取。

## 5. API

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/snapshot` | 全量快照（前端 5s 轮询） |
| GET | `/api/history?host=<id>&range=6h` | 主机历史曲线，`range` 支持 `1h` / `6h` / `24h` / `7d` |
| GET | `/api/health` | 自身健康检查 |
| GET | `/` | 静态前端 |

`/api/snapshot` 响应（示意，字段省略了 `labels`/`description` 等可选值）：

```json
{
  "ts": 1730000000,
  "hosts": [{
    "id": "web-01", "name": "web-01", "kind": "linux", "address": "192.168.1.10",
    "online": true, "last_seen": 1730000000,
    "cpu": 12.4, "mem_used": 3221225472, "mem_total": 8589934592,
    "disk_used": 100000000000, "disk_total": 200000000000,
    "load1": 0.42, "uptime": 123456, "temp": 42.0, "kernel": "6.12.0",
    "net_rx": 6061, "net_tx": 8578,
    "cpu_hist": [1.2, 3.4, 2.1]
  }],
  "containers": [{
    "host": "web-01", "name": "nginx", "image": "nginx:1.27",
    "state": "running", "status": "Up 2 hours (healthy)",
    "cpu": 1.2, "mem": 102400000, "mem_limit": 536870912, "restarts": 0
  }],
  "pve": {
    "nodes": [{"name": "pve", "online": true, "cpu": 8.1, "mem_used": 28933832704, "mem_total": 33575600128, "uptime": 5635848}],
    "guests": [{"node": "pve", "id": 100, "name": "vm-100", "kind": "qemu", "status": "running", "cpu": 3.2, "mem": 0, "maxmem": 0, "disk": 0, "maxdisk": 0}]
  },
  "services": [{"name": "NAS", "url": "http://192.168.1.5/", "ok": true, "code": 200, "latency_ms": 12, "cert_days": null}],
  "alerts": [{"level": "error", "text": "web-01 离线 32s"}]
}
```

### 字段说明

`hosts[]`

| 字段 | 说明 |
| --- | --- |
| `kind` | `linux` / `macos` / `ikuai`，前端据此切换图标与指标 |
| `cpu_hist` | 最近 36 次 CPU 采样（10s 一次 ≈ 6 分钟），供 sparkline |
| `net_rx` / `net_tx` | 字节/秒，macOS 与 iKuai 不提供（iKuai 用 `wan_*`） |
| `clients` / `conns` / `wan_up` / `wan_down` / `wan_ip` / `wan_status` | 仅 iKuai：在线终端、连接数、WAN 上下行、WAN IP、线路状态 |
| `err` | 采集失败原因（离线时） |

`pve.guests[]` 用 `kind` 区分 `qemu` / `lxc`；`node` 是节点显示名（多台 PVE 时用配置的 `name`）。

`alerts[]` 只有 `level`（`error` / `warn`）和 `text`，由后端 `buildAlerts()` 生成。

### 历史接口

```json
{"host": "web-01", "range": "6h", "samples": [{"ts": 1730000000, "cpu": 12.4, "mem_used": 1, "mem_total": 2, "disk_used": 3, "disk_total": 4, "load1": 0.4}]}
```

## 6. 前端

六页，1024×768 基准，每页一屏铺满。**每屏只回答一个问题**，同一指标不在两页重复。

| 页 | 只回答 | 布局 |
| --- | --- | --- |
| 概览 | 有没有问题？ | 4 个 KPI（主机 / 容器 / 虚拟机 / 服务在线数）+ 告警列表；无告警时整块 ALL SYSTEMS NOMINAL |
| 主机 | 每台机器怎么样？ | 2 列卡片：图标 / 名称地址 / CPU 6 分钟 sparkline / CPU·MEM·LOAD·DISK / 网卡上下行 |
| 容器 | 哪些容器活着、在哪台机器上？ | 6 列等格；每格带**主机名标签**（主机强调色），左侧色条跟随主机，同机容器颜色一致 |
| 虚拟化 | PVE 节点与客户机状态 | 上 36% 节点格（2 列），下 64% 客户机格（4 列紧凑版：CPU/MEM 条 + RAM/DISK 脚注） |
| 服务 | 外部服务通不通？ | ≤4 个 2 列，否则 3 列：大号响应时间 + ONLINE/DOWN + 状态码/TLS 天数 |
| 趋势 | 过去 6 小时走势 | 每台主机一张 6h 曲线（CPU 强调色、MEM 青绿、90% 阈值虚线） |

轮播 20s/页；手指左右拖动跟手滑动，松手按位移/速度决定翻页或回弹；完成一次滑动后轮播暂停 60s。
普通点击不暂停轮播。

### 状态强调
- 运行中：正常底色 + 主机色左边条（容器）/ 类别色图标。
- **停机容器**：45° 斜线纹理底 + 灰标题 + 空心标记，整屏扫一眼就能看出"死掉的那一片"。
- 警告 / 异常机器：左侧 **4px** 状态条 + 描边 + 同色底 + ▲/✕ 标记，颜色取自当前主题的
  `warn` / `crit`（例如 Mocha 是琥珀/珊瑚红，Gruvbox 是橙/红，Latte 是暗金/深红）。
- 离线机器：降饱和灰底 + ○ 空心标记。

### 选择页（首次进入）
打开页面先显示外观选择页：上半是 5 套配色（带迷你预览），下半是 5 种背景（实时预览），
底部有倒计时自动继续。选择写入 `localStorage`（`paddeck.theme` / `paddeck.bg`），
想换配色或背景重新打开本页即可，不需要设置按钮。
URL 参数 `?theme=<id>&bg=<id>` 可直接指定并跳过选择页（书签/自动化用）。

### 背景系统
| 名称 | 实现 | 成本 |
| --- | --- | --- |
| 极光 AURORA | 4 层大尺寸径向渐变 | 静态绘制，零运行时成本 |
| 弥散 MESH | 4 角径向渐变 + 深色压角 | 静态 |
| 点阵 DOT | `radial-gradient(circle, …)` + `background-size: 26px` | 静态、平铺 |
| 网格 RETRO | `rotateX(64deg)` 透视网格 + 14s 平移 | 合成层位移 |
| 纯色 NONE | 无 | 零 |

全部不依赖 canvas、`filter` 或混合模式——iPad 4 的 A6X 上没有余量。

### 启动页（无数据）
首次加载或后端重启后第一份快照到达前，覆盖一层启动页：双环旋转 SVG + `SYNCING` +
极光背景，数据到达自动淡出。此时不显示「NO DATA」。

### 动效系统
| 效果 | 实现 | 时机 |
| --- | --- | --- |
| 翻页 | `translate3d` 双页滑动（0.34s 缓动曲线），只有出/入两页在渲染树上 | 翻页 / 拖拽 |
| 卡片边框流光 | 卡片内两条渐变彗星，只跑 `translate3d` | 常驻 |
| 顶部光轨 | 每张大卡顶部 2px 强调色渐变（流光从上面划过） | 静态 + 流光 |
| 聚光 spotlight | 560px 径向渐变层跟随手指位移，触摸时淡入 | 触摸 |
| 倾斜 tilt | `perspective(720px) rotateX/Y ≤3.2°`，松手 0.3s 回正 | 按住拖动 |
| Shimmer | 一条 106° 斜向光带 9s 扫过全屏 | 常驻背景 |
| 3D 数字 | 多层 `text-shadow`（亮面 + 暗面 + 光晕），阴影色随主题生成 | 静态 |
| 荧光划线 | `brand` 下方 2px 高亮线 + 发光、页签左侧竖条、当前页方块指示点 | 静态 |
| CPU 趋势 | 主机卡内嵌 6 分钟 sparkline，纵轴自动缩放（峰值 ×1.25，最低 25%），≥90% 才画阈值线 | 每 5s |

### 自适应降级
`requestAnimationFrame` 采样帧间隔；连续 45 帧 > 42ms（<24fps）时给 `<body>` 加 `.lowfx`：
停掉网格漂移、shimmer、流光、聚光，卡片过渡也去掉。也可以用 `?lowfx` 手动强制降级档。
iPad 4 上如果出现掉帧会自动退回静态模式，不会出现"动画拖垮整块面板"。

### 原地更新
每 5s 的轮询不重建整页：按 id/名称生成结构 key，key 不变时只改文本、宽度和颜色属性。
这样流光、倾斜、sparkline 不会每 5s 被打断重来。

### 资源生成（主题 / 背景）
Safari 10 没有 CSS 变量，所以每套主题都是一份**完整样式表**：

```
web/style.css              # 母版，Mocha 调色板 + 全部布局
tools/themes.py            # 从母版按调色板渲染出 N 份主题
web/theme-<id>.css         # 生成物（提交进仓库）
web/themes.js              # 生成物：主题元数据 + 前端用色（picker/JS 常量）
```

切换主题 = 换 `<link id="theme">` 的 href。新增一套主题只需在 `tools/themes.py` 里
加一个 `theme(...)` 调用，然后 `python3 tools/themes.py`。

### 兼容性约束
- 语法：ES5（`var`、`function`、无箭头函数/模板字符串/解构/async）
- 请求：`XMLHttpRequest`
- 布局：flex + 百分比宽高（JS 计算单元格尺寸），不使用 CSS Grid / CSS 变量
- 图表：手写 SVG `polyline`
- 图标：内联 `<svg>`（Lucide 风格 24×24 描边路径），按名称关键词匹配
- 字体：系统字体 + `Menlo` 等宽数字
- 无构建步骤，`go:embed` 直接托管
- 主屏模式（`navigator.standalone`）下顶栏额外留 20px 给 iOS 状态栏

### 视觉 token（以 Mocha 为例，其余主题按同一张表生成）

| 用途 | Token | 色值 |
| --- | --- | --- |
| 画布 | Base | `#1e1e2e` |
| 面板 | Surface 0 | `#313244` |
| 边框 | Surface 1 | `#45475a` |
| 发丝线 | — | `#3d3f54` |
| 正文 | Text | `#cdd6f4` |
| 次级 | Subtext 0 | `#a6adc8` |
| 弱化 | Overlay 0 | `#6c7086` |
| 强调 1–6 | Mauve / Teal / Pink / Lavender / Peach / Sky | `#cba6f7` `#94e2d5` `#f5c2e7` `#b4befe` `#fab387` `#89dceb` |
| 正常 / 警告 / 异常 | Green / Yellow / Red | `#a6e3a1` / `#f9e2af` / `#f38ba8` |

- 每页一个强调色：概览 Mauve、主机 Teal、容器 Pink（容器格改用主机色）、虚拟化 Lavender、
  服务 Peach、趋势 Sky；页签文字、当前页指示点、卡片顶部光轨都跟着变。
- 图标常态按类别着色，异常时才切成主题的 warn / crit 色。
- 状态形状冗余：● 正常 / ▲ 警告 / ✕ 异常 / ○ 离线。
- 触摸热区 ≥44px。

## 7. 容错
- 单个采集器失败不影响其他目标。
- 失败保留上一次数据，标记 `stale` 与 `last_seen`。
- 前端请求失败保留旧数据，顶部显示「连接中断」。
- 后端启动时若无配置则拒绝启动并打印明确错误。

## 8. 部署
- 镜像：`FROM golang:1.24-alpine AS build` → `FROM alpine:3.20`
- 卷：`/data`（SQLite）、`/keys`（只读 SSH key）
- 端口：9080
- 健康检查：`/api/health`

## 9. iPad Kiosk
1. Safari 打开面板地址（可带 `?theme=…&bg=…` 跳过外观选择页），分享 → 添加到主屏幕。
2. 设置 → 显示与亮度 → 自动锁定 → **永不**。
3. 设置 → 辅助功能 → 引导式访问 → 开启；进入面板后三击侧边键锁定。
4. 插电，亮度 40–50%（老屏幕长期满亮度会加速老化）。

已知限制：**iOS 10 无法隐藏顶部状态栏**（时间/WiFi/电池）。`black-translucent` 只让网页内容
延伸到状态栏下方，无法真正全屏；面板在 `navigator.standalone` 模式下给顶栏额外留 20px，
避免时钟和电池图标重叠。要彻底去掉状态栏只能做原生壳（WKWebView + `prefersStatusBarHidden`）。

## 10. 开源
- MIT
- README 中英文
- 配置示例脱敏
- CI：`go vet`、`go test`、交叉编译
