# 配置说明

PadDeck 的配置只有一个 YAML 文件。**所有段落都是可选的**——只配一台机器、只配几个网址，
甚至只配一个路由器都能跑。没有的设备类型直接不写。

```bash
paddeck -init            # 生成带注释的最小配置
vi config.yaml
paddeck -check           # 逐项体检
paddeck
```

## 配置文件在哪

不传 `-config` 时按顺序查找，找到一个就用：

1. `$PADDECK_CONFIG`
2. `./config.yaml`
3. `./config.local.yaml`
4. `$XDG_CONFIG_HOME/paddeck/config.yaml`（默认 `~/.config/paddeck/config.yaml`）
5. `/config/config.yaml`（Docker 镜像里的位置）

数据库同理：`-db` → `$PADDECK_DB` → `/data/paddeck.db`（存在 `/data` 时）→ `./data/paddeck.db`。

## 环境变量

配置文件里任意字符串值支持 `${VAR}` 与 `${VAR:-默认值}` 展开，适合把密码放在环境变量里：

```yaml
hosts:
  - id: router
    type: ikuai
    address: 192.168.1.1
    username: admin
    password: ${IKUAI_PASSWORD}
```

## 顶层结构

```yaml
server:
  listen: ":9080"

collector:
  interval: 10s          # 主机采集间隔
  docker_interval: 30s   # Docker 采集间隔（docker stats 较重）
  history_interval: 60s  # 历史写入间隔
  http_timeout: 3s       # HTTP 探活超时
  ssh_timeout: 8s        # SSH 超时
  retention_days: 30     # 历史保留天数
  ssh_key: ~/.ssh/id_ed25519  # 可省略，省略时自动查找

hosts:      # 任意台：Linux / macOS / 爱快
pve:        # 任意台 Proxmox VE 节点
services:   # 任意个 HTTP 端点
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

## hosts

### Linux（默认）

```yaml
hosts:
  - id: web-01
    name: web-01            # 显示名，省略时用 id
    address: 192.168.1.10
    user: youruser
    port: 22
    docker: true            # 采集容器
    docker_sudo: false      # docker 需要 sudo -n 时打开
    docker_path: docker     # 群晖是 /usr/local/bin/docker
    disk_mounts: ["/"]      # 磁盘统计的挂载点
    labels: ["web", "debian"]
```

采集内容：CPU、内存、磁盘、负载、运行时长、温度、网卡上下行。

### macOS

```yaml
  - id: mac-01
    type: macos
    address: 192.168.1.20
    user: youruser
```

需要在「系统设置 → 通用 → 共享 → 远程登录」打开 SSH。
采集内容：CPU（`top`）、内存（`vm_stat`）、磁盘（`df`）、负载、运行时长。macOS 暂不采集网卡速率与温度。

### 爱快路由（Web API，不需要 SSH）

```yaml
  - id: router
    type: ikuai
    address: 192.168.1.1
    api_url: https://192.168.1.1:444   # 可省略，默认 https://<address>:444
    username: admin
    password: ${IKUAI_PASSWORD}
```

采集内容：CPU、内存、在线终端数、连接数、WAN 上下行速率、WAN IP、线路状态、运行时长。

### 通用：任何设备都能探活

没有 SSH 也没有 API 的设备（交换机、摄像头、打印机、路由器后台……），
用 `services` 里的 HTTP 探活即可，见下文。

## pve

```yaml
pve:
  - id: pve
    name: pve
    address: 192.168.1.4
    user: root
```

需要 root SSH（采集靠 `pvesh`）。可以列多台；两台独立 PVE 的节点都叫 `pve` 时，
面板会显示成配置里的 `name`。

## services

```yaml
services:
  - name: NAS
    url: http://192.168.1.20/
  - name: Homepage
    url: https://home.example.com
```

- 2xx/3xx 视为健康，4xx/5xx 与超时视为不可用
- 记录状态码、延迟、TLS 证书剩余天数
- 这是**唯一适用于所有设备**的采集方式，不需要任何凭据

## 告警

内置规则（阈值目前写死，计划改为可配置）：

| 条件 | 级别 |
| --- | --- |
| 主机离线 | error |
| CPU ≥ 90% | warn |
| 内存 ≥ 90% | warn |
| 磁盘 ≥ 90% | warn |
| 服务探活失败 | error |

## 校验与排错

```bash
paddeck -check
```

会逐台探测 SSH、docker、pvesh、爱快登录、HTTP，失败时给出修复提示。常见问题：

| 现象 | 原因 |
| --- | --- |
| `no SSH credentials` | 没有可用的 key：`ssh-copy-id`，或设置 `collector.ssh_key`，或启动 ssh-agent |
| `ssh handshake` | 用户/端口不对，或目标没开 SSH |
| `docker: command not found` | 设置 `docker_path`，或用户没有 docker 权限（试 `docker_sudo: true`） |
| 爱快 `login failed` | 账号密码错，或 `api_url` 端口不是 444 |
| PVE `pvesh` 报错 | 需要 root，且是 Proxmox VE 6+ |
| 温度 / 网卡显示 `--` | 该设备没有上报这个指标（macOS 无网卡速率，虚拟机无温度传感器） |
