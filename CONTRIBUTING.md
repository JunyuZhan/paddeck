# 贡献指南

感谢你愿意改进 PadDeck。这个项目的核心约束是**兼容 iOS 10 / Safari 10 的老设备**，
所以「能不能在老 iPad 上流畅跑」优先于「代码写得新不新」。

## 环境

- Go 1.24+
- Python 3.9+（只有改配色时需要，用来跑 `tools/themes.py`）
- 不需要 Node / 打包器

```bash
git clone https://github.com/JunyuZhan/paddeck.git
cd paddeck
go build -o paddeck ./cmd/paddeck
./paddeck -init && vi config.yaml && ./paddeck -check && ./paddeck
```

详细开发说明见 [docs/development.md](docs/development.md)。

## 提交前自查

```bash
gofmt -l .        # 应为空
go vet ./...
go test ./...
python3 tools/themes.py && git diff --exit-code -- web/theme-*.css web/themes.js
```

如果改了 `web/style.css` 里的颜色，**必须**重新生成主题文件并一起提交；
CI 会校验主题文件与母版是否同步。

## 代码约定

- **Go**：标准库优先，错误信息写清楚「哪个主机 / 哪个字段」，不要裸 `panic`
- **前端**：ES5 语法（`var` / `function`），不用 CSS 变量、CSS Grid、`fetch`、`async`；
  动效只用 `translate3d` 和静态渐变，避免整屏重绘
- **采集器**：新增设备类型时先写解析测试，再接入 `manager`
- **隐私**：不要提交真实 IP、主机名、用户名、容器名、密钥。
  示例统一用 `192.168.1.x`、`web-01`、`youruser`

## 提交信息

`type(scope): subject`，type 取 `feat` / `fix` / `docs` / `style` / `refactor` / `test` / `chore`。
示例：`fix(collector): ignore docker header line`

## Pull Request

1. 从 `main` 切分支：`feat/xxx`、`fix/xxx`
2. 改动尽量聚焦一个主题，附上「为什么」
3. 涉及 UI 的请附截图（1024×768）
4. 涉及采集器的请说明在哪种设备/版本上验证过
5. 确保 CI 全绿

## 行为准则

参与本项目即表示同意遵守 [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)。

## 报 Bug

请用 Issue 模板，附上：

- PadDeck 版本（`paddeck -version`）与部署方式
- 操作系统 / iPad 型号 / 浏览器版本
- `paddeck -check` 的输出（**记得把内网 IP 打码**）
- 相关日志（`docker logs paddeck`）
