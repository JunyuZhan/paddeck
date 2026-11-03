# PadDeck 开发计划

当前版本 **v0.2**（本地 git，尚未推 GitHub）。本文档记录已完成、下一步、以及被否决的想法，
避免以后重复讨论。设计细节见 [design.md](design.md)，开发方式见 [development.md](development.md)。

---

## 1. 已完成

### v0.1 — 能用
- [x] Go 单二进制 + `go:embed` 前端 + Docker 部署
- [x] SSH 采集 Linux 主机（CPU/内存/磁盘/负载/温度/运行时长）
- [x] Docker 容器列表与资源占用
- [x] Proxmox 节点与 QEMU/LXC 客户机
- [x] HTTP 探活（状态码/延迟/TLS 剩余天数）
- [x] SQLite 历史（1 分钟聚合、30 天保留、查询降采样）
- [x] 6 页轮播 + 触摸滑动 + iPad 主屏 kiosk 适配
- [x] 告警规则（离线 / CPU≥90 / 内存≥90 / 磁盘≥90 / 服务不可用）

### v0.2 — 好看、好看清
- [x] NOC 风格重做：全屏等格图标网格、6 页一屏、形状冗余状态标记
- [x] 爱快路由采集（Web API 登录 + sysstat + WAN 状态）
- [x] 第二台 PVE（节点名用配置名区分）
- [x] macOS 采集器（top / vm_stat / sysctl / df），Mac mini 接入
- [x] Linux 网卡上下行速率（`/proc/net/dev` 差值）
- [x] 5 套配色主题 + 5 种背景，进入时选择、`localStorage` 记忆
- [x] 动效：翻页滑动、跟手拖动、边框流光、聚光、倾斜、shimmer、3D 数字、CPU sparkline
- [x] 自适应降级 `.lowfx`（rAF 帧率探针 + `?lowfx` 手动开关）
- [x] 原地更新（5s 轮询不重建 DOM，不打断动画）
- [x] 页面职责重排：概览只做 KPI+告警，主机页才铺资源细节
- [x] 容器标注所属主机（主机色标签 + 左边条），停机容器斜线纹理
- [x] 无数据启动页（双环 spinner + SYNCING）
- [x] 删掉 scramble 乱码动效
- [x] 主屏模式顶栏留 20px 给 iOS 状态栏

### 修过的坑（别再踩）
- 跟手滑动完全失效：`drag.start` 存的是 X 坐标却拿去算 `dy`，手势永远被判成滚动
  → 起点拆成 `startX/startY`；`touchmove` 一律 `preventDefault` 防 iOS 抢手势
- 触摸一次轮播冻结 2 分钟 → 只有真正滑动完成才暂停 60s
- `render()` 抛异常被 XHR 的 `try/catch` 吞掉，误显示「连接中断」→ 渲染单独 try/catch 并只打 console
- `.hcard-spark` 里 SVG 只有 30px 宽 → 父容器要 `display:block` 而不是 flex

---

## 2. 下一步（P0，按优先顺序）

### 2.1 视觉风格第二批
用户候选清单里剩下的、且 iPad 4 扛得住的：

| 风格 | 实现设想 | 成本 |
| --- | --- | --- |
| 噪点 GRAIN | 一张 4×4 噪声 PNG（data URI）平铺 + `opacity .04` 叠在所有背景上 | 零（静态图） |
| 星空 STARFIELD | 两层静态星点（`box-shadow` 列表或 SVG 圆点），慢速 `translate3d` 视差 | 低（合成层位移） |
| 字符雨 MATRIX | canvas 逐帧绘字；**只在 `.lowfx` 关闭时启用**，并限制列数/帧率 | 高（A6X 风险） |
| ASCII 艺术 | 仅用于启动页/空数据页的静态文字画 | 零 |

验收标准：iPad 4 上三页轮播 30 分钟不掉帧、不发热卡顿；`?lowfx` 下全部关闭。

### 2.2 容器页按主机分组
目前容器格按 `host` 排序但视觉上仍是一整片。计划：按主机分块，每块一个主机标题条
（主机色 + 名称 + 数量），块内仍是等格。好处是"一台机器挂了"能一眼定位。

### 2.3 告警阈值可配置
`buildAlerts()` 里 CPU/内存/磁盘 90% 是写死的。计划移到 `collector:` 配置：

```yaml
collector:
  alert:
    cpu: 90
    mem: 85
    disk: 90
```

### 2.4 选择页体验
- [x] 支持 `?theme=gruvbox&bg=aurora` 直接跳过选择页（书签/自动化）
- [ ] 记住上次选择后，倒计时从 15s 缩短到 5s（kiosk 重启用）
- [ ] 选择页也应用当前背景的实时动画（现在预览是静态的）

### 2.5 开源发布
- [x] 隐私清理：内网 IP / 主机名 / 用户名 / 容器名全部替换为通用示例
- [x] 历史重写：公开仓库只保留一个干净的初始提交（旧历史留在本地 `archive/pre-public`）
- [x] 标准开源文件：LICENSE / CONTRIBUTING / SECURITY / CODE_OF_CONDUCT / CHANGELOG / Makefile
- [x] Issue / PR 模板、CI（gofmt / vet / test / build / 私密文件检查 / 主题同步）
- [ ] README 加截图（需要先补 `docs/screenshots/`）
- [ ] 打 tag `v0.3.0` 并 publish release

### 2.6 开箱即用
- [x] `paddeck -init` 生成最小配置
- [x] `paddeck -check` 逐项体检（SSH / docker / pvesh / 爱快 / HTTP）
- [x] 配置自动发现（5 个位置）+ `PADDECK_CONFIG` 环境变量
- [x] 配置 `${ENV}` 展开
- [x] SSH 密钥自动查找（`~/.ssh/id_*` / ssh-agent）
- [ ] `paddeck -demo`：内置假数据，`docker run` 即可看到界面（评估用）
- [ ] 首次启动时若没有配置，直接打印 `-init` 指引（现在会报错并提示）

---

## 3. 中期（P1）

| 功能 | 说明 | 价值 |
| --- | --- | --- |
| 通知推送 | 告警发 ntfy / Telegram / 企业微信 | 面板只能"看得到"，推送才能"叫醒人" |
| 主机分组 | 按 `labels` 分组（业务/存储/路由），概览按组聚合 | 设备变多后必需 |
| 容器详情页 | 点开单个容器看 CPU/内存曲线、重启次数、镜像 | 现在只有一屏概览 |
| 服务探活历史 | 记录每次探测结果，画出可用率/延迟曲线 | 趋势页现在只有主机指标 |
| API 只读鉴权 | 可选 token，防止面板暴露在公网被扫 | 上公网的前提 |
| 主题/背景保存到服务端 | 现在存在 iPad 的 localStorage，换设备要重选 | 多设备一致性 |
| 工程化 | 前端拆文件（`app.js` 已 1300 行）、加端到端冒烟测试 | 可维护性 |

## 4. 远期（P2，不一定做）

- Windows / 群晖 DSM / 交换机（SNMP）采集器
- Prometheus 导出（`/metrics`），让 PadDeck 变成采集网关
- WebSocket 推送替代 5s 轮询（Safari 10 支持，但需要后端长连接管理）
- 多语言（中/英）
- 移动端竖屏布局（现在只针对 1024×768 横屏优化）
- 音效/语音播报告警（kiosk 场景可选）

---

## 5. 视觉风格候选清单（13 项）评估

用户提出的 13 种风格，逐项判断。**判断依据：iPad 4（A6X，2012）+ Safari 10 的实机表现。**

| # | 风格 | 状态 | 结论 |
| --- | --- | --- | --- |
| 1 | 极光 aurora | ✅ 已上线 | 4 层径向渐变，零运行时成本 |
| 2 | 流动渐变 mesh gradient | ✅ 已上线（静态版） | 真正"流动"需要动画整屏背景 → 重绘，没做 |
| 3 | 粒子波浪 particle wave | ❌ 不做 | 全屏 canvas 粒子，A6X 大概率 <20fps |
| 4 | 点阵地球 dotted globe | ❌ 不做 | 需要 WebGL/canvas 3D 投影 |
| 5 | 流体 fluid | ❌ 不做 | WebGL shader 才好看，iOS 10 不可控 |
| 6 | 光束 beams | ✅ 已有 | 卡片顶部光轨 + 全屏 shimmer |
| 7 | 字符雨 matrix rain | ⏳ 下一批 | canvas 逐帧写字符，需限列数/帧率，且只在非 lowfx 下开 |
| 8 | ASCII 艺术 | ⏳ 下一批（限启动页） | 当背景干扰读数，只放空数据页 |
| 9 | 点阵背景 dot grid | ✅ 已上线 | `radial-gradient` 平铺 |
| 10 | 透视网格 retro grid | ✅ 已上线 | `rotateX` + 合成层平移 |
| 11 | 星空穿梭 starfield | ⏳ 下一批 | 静态星点 + 视差位移，不用 canvas |
| 12 | 噪点颗粒 grain | ⏳ 下一批（首选） | 静态噪声图叠加，零成本，质感提升明显 |
| 13 | 渐变背景 gradient | ✅ 已被 1/2 覆盖 | — |

**取舍原则**：任何需要每帧重绘全屏像素的方案（canvas / filter / 大面积动画）一律否决或
放进 `.lowfx` 白名单。面板是 7×24 常亮的，发热和掉帧比炫酷重要。

---

## 6. 已知问题 / 技术债

| 问题 | 影响 | 计划 |
| --- | --- | --- |
| 告警阈值写死 90% | 不同机器合理水位不同 | P0 2.3 |
| `app.js` 单文件 ~1300 行 | 改动容易踩到别处 | P1 拆模块（不用打包器，`<script>` 顺序加载） |
| PVE/Docker 采集每轮都跑，超时 20s | 慢节点会拖长整轮 | 加并发上限与失败退避 |
| 爱快会话过期只重试一次 | 路由重启后可能连续失败一轮 | 加重登退避 |
| macOS 无网卡速率、无温度 | 主机卡片显示 `--` | 需要 `netstat -ib` 解析 / 需要 sudo，暂缓 |
| 选择页每次打开都出现 | kiosk 重启会停在选择页 15s | P0 2.4 缩短倒计时 + URL 跳过 |
| iOS 状态栏无法隐藏 | 顶部 20px 永远被占 | 平台限制，只能留白；要真全屏得做原生壳 |
| 无鉴权 | 只能内网使用 | P1 |
| 没有端到端测试 | 前端回归靠肉眼 | P1 加 Puppeteer/Chromium 冒烟脚本（仅开发机） |
| 主题必须重新生成 | 改 `style.css` 颜色后忘记跑 `tools/themes.py` 会导致主题错色 | 在 CI 里加一条校验：重新生成后 `git diff --exit-code` |

---

## 7. 决策记录（ADR）

### ADR-1 不用 CSS 变量，改为"生成式主题文件"
Safari 10 不支持 `var(--x)`。备选：① 每套主题手写 override ② 生成完整样式表。
选 ②：`tools/themes.py` 从一份母版渲染，保证没有漏改的颜色，代价是新主题要跑脚本。

### ADR-2 背景不用 canvas
iPad 4 的 GPU 是 PowerVR SGX554MP4，全屏 canvas 动画会掉到 15–20fps 并明显发热。
所有背景只用静态渐变/图案；唯一常驻动画是合成层位移（流光、shimmer、网格漂移）。

### ADR-3 轮询用"原地更新"而不是重建 DOM
每 5s `innerHTML = …` 会把流光、倾斜、sparkline 全部重置，视觉上一直在闪。
改为结构 key + 局部更新，代价是要为每页写 `build` 和 `update` 两个函数。

### ADR-4 外观选择放在"进入时"，不做设置按钮
用户提出：与其在面板里加设置入口，不如每次打开时选一次，选完记住。
好处是零交互成本、不占用面板空间；代价是 kiosk 重启会多一次选择（用倒计时缓解）。

### ADR-5 一屏一用，指标不跨页重复
最初概览和主机页都显示 CPU/内存/磁盘，属于重复。现在概览只回答"有没有问题"，
主机页才回答"每台机器怎么样"，其余页面各自独立。

### ADR-6 状态色跟主题走，不写死红色
"停机/越界要醒目"但用户要求"红色按配色来"。所以所有状态色都映射到主题的
`warn` / `crit` token，换主题时告警色自动协调（例如 Gruvbox 是橙红，Latte 是暗红）。
