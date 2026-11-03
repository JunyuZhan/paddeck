# 安全说明

## 报告漏洞

请不要开公开 Issue。通过以下任一方式私下联系：

- GitHub Security Advisories：仓库 **Security → Report a vulnerability**
- 邮件：`junyuzhan@users.noreply.github.com`

请在报告里说明：影响版本、复现步骤、可能的后果。48 小时内会收到确认，
修复发布后会在 [CHANGELOG](CHANGELOG.md) 中致谢（除非你要求匿名）。

## 支持范围

| 版本 | 支持 |
| --- | --- |
| 0.3.x | ✅ 是当前版本 |
| < 0.3 | ❌ 未公开，不提供支持 |

## 安全模型（重要）

PadDeck 是**内网只读面板，没有登录、没有鉴权、没有 TLS**。这是有意的设计取舍：
它的目标是在可信局域网里用一台老 iPad 看状态，任何暴露到公网的行为都会带来风险。

请务必：

- 只在内网 / VPN 内使用；需要公网访问时套一层反向代理 + 认证（如 Authelia、Cloudflare Access）
- 用**独立的只读 SSH 密钥**，不要复用个人密钥；建议在 `authorized_keys` 里加
  `command=` 或 `restrict` 限制（PVE/爱快需要采集多类信息，通常不建议限制过死）
- 通过 `${ENV_VAR}` 注入密码，不要把密码写进 `config.yaml`
- `config.yaml`、`keys/`、`data/` 不要提交到任何仓库（已在 `.gitignore`）

## 已知风险

| 项 | 说明 |
| --- | --- |
| 无鉴权 | 能访问 9080 端口的人都能看到全部指标 |
| SSH HostKey 未校验 | 采集器为了兼容自签/重装环境忽略 host key（`InsecureIgnoreHostKey`）；如果在意，请把采集机限制在受控网段 |
| 配置里可能出现明文密码 | 用环境变量可避免 |
| 自签证书 | 爱快 / PVE 采集默认跳过 TLS 校验 |
