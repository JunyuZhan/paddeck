# 用 AI Agent 部署 PadDeck

> 不用自己敲命令：把下面这段提示词丢给会执行命令的 AI 编程助手（Claude Code、Cursor、
> Codex、Aider、OpenCode 等），它会检查你的环境、问你有哪几台设备、写好配置、
> 跑体检、把失败项修好，最后把地址给你。

原因是每个人的环境都不一样：有人只有一台服务器，有人是 NAS + 软路由，有人用 PVE 集群，
有人根本没有任何 Linux 机器、只有几个网页服务。让 agent 按实际情况组装，比让用户对着
一份「别人家的配置」改要靠谱得多。

## 用法

1. 在要运行 PadDeck 的机器上，用你的 AI 助手打开一个终端会话
2. 把下面前两段（提示词 + 你的情况）粘贴进去
3. 跟着它的提问回答就行

## 提示词

```text
帮我在这台机器上部署 PadDeck（一个局域网 Homelab 监控面板，Go 单二进制，仓库：
https://github.com/JunyuZhan/paddeck）。请按以下步骤做，每步都先说明你要做什么：

1. 检查环境：go 版本（需要 1.24+）、是否有 docker、这台机器的 IP、能否访问 github。
   如果缺少 go 且没有 docker，先告诉我安装方案，不要擅自 sudo 安装。
2. 获取代码：git clone 仓库（或下载 release 里的二进制）。
3. 问我有哪些监控目标，一项一项问，不要假设：
   a. 有哪些 Linux 主机（hostname / IP / SSH 用户名，是否装了 docker）
   b. 有没有 macOS / 群晖 / 其它 NAS（能不能 SSH）
   c. 有没有 Proxmox VE（几台，root SSH 是否可用）
   d. 有没有爱快路由器（iKuai，Web API 管理口地址与账号）
   e. 有哪些想监控的网址（面板、NAS 后台、路由器后台……）
   我可能只有其中一部分，按实际情况来。
4. 帮我准备只读 SSH 密钥（如果我要用 SSH 采集）：生成专用 key，用 ssh-copy-id 分发，
   并验证 ssh 能连通。不要把私钥写进任何会被提交的文件。
5. 生成 config.yaml：只用我确认过的目标；密码走 ${ENV_VAR} 而不是明文；并告诉我这些
   环境变量怎么在 systemd / docker 里设置。
6. 运行 `./paddeck -check`，把 FAIL 的项逐条分析原因并修复（网络、权限、路径、sudo），
   直到全部 ok；如果某项确实不可用，跟我确认是删掉还是保留为已知失败。
7. 启动：用 docker compose 或 systemd 常驻（重启后自动拉起），然后验证
   /api/health 与 /api/snapshot 有数据。
8. 最后告诉我：
   - 面板地址（http://<IP>:9080）
   - iPad 添加到主屏的步骤（我的是老 iPad，iOS 10 / Safari 10）
   - 用 `./paddeck -check` 复查的命令
   - 配置文件和密钥放在哪，哪些不能提交到 git
```

## 补充说明我的情况（示例）

```text
我的情况：一台 PVE 主机（root SSH 可用），上面跑着 6 个 LXC；一台群晖 NAS（可以 SSH，
docker 在 /usr/local/bin/docker）；一台爱快软路由（管理地址 192.168.1.1，账号 admin）；
另外想监控 Home Assistant 和 Grafana 两个网址。没有 macOS。
```

## 没有设备？先用演示模式看效果

```bash
./paddeck -demo
# 打开 http://localhost:9080 —— 合成数据，不连任何真实设备，不需要配置文件
```

## 用它来验证 agent 是否做对了

```bash
./paddeck -check        # 每一项都应该是 ok
curl -s localhost:9080/api/health
curl -s localhost:9080/api/snapshot | head -c 300
```

## 安全提醒

- 让 agent 生成密钥时，用**专用只读密钥**，不要复用你个人的 `~/.ssh/id_rsa`
- 密码请让它写 `${IKUAI_PASSWORD}` 之类的占位符，再由 systemd `Environment=` 或
  docker `env_file` 注入；不要出现在 `config.yaml` 里
- 面板本身没有登录，只在内网用；要暴露到公网请套反向代理 + 认证
