# 变更记录

本项目遵循 [语义化版本](https://semver.org/lang/zh-CN/)，格式参考 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)。

## [Unreleased]

### 计划中
- 背景风格第二批：噪点 grain、星空 starfield、字符雨 matrix（可关）
- 容器页按主机分组
- 告警阈值可配置（CPU / 内存 / 磁盘）

## [0.3.0] - 2026-11-03

首个公开版本。

### 新增
- 6 页 NOC 面板：概览（KPI + 告警）、主机、容器、虚拟化、服务、趋势
- 5 套配色（Catppuccin Mocha、Rosé Pine Moon、Gruvbox Dark、Nord、Catppuccin Latte）与 5 种背景
  （极光、弥散渐变、点阵、透视网格、纯色），进入页面时选择，支持 `?theme=&bg=` 跳过
- 容器格标注所属机器（主机色标签 + 左边条），停机容器斜线纹理
- 主机卡片：CPU 6 分钟 sparkline、CPU/MEM/LOAD/DISK、网卡上下行
- 爱快路由采集（Web API 登录 + sysstat + WAN 状态）
- macOS 采集器（top / vm_stat / sysctl / df）
- Linux 网卡速率（`/proc/net/dev` 差值）
- 多台 PVE，节点名取配置里的 `name`
- 无数据启动页（SYNCING + 双环动画）
- `-init` 生成配置、`-check` 逐项体检、配置自动发现、`${ENV}` 展开、SSH 密钥自动查找
- 自适应降级 `.lowfx` 与 `?lowfx` 手动开关
- Docker 部署（多阶段构建、健康检查）

### 变更
- 概览页只展示状态与告警，主机细节移到主机页，避免指标跨页重复
- 轮播暂停规则：完成一次滑动后暂停 60 秒（此前任何触摸都暂停 2 分钟）
- 主屏模式下顶栏额外留 20px 给 iOS 状态栏

### 修复
- 手势滑动完全失效：起点坐标被同时用于 X/Y 判断，导致永远被判成纵向滚动
- 渲染异常被 XHR 的 try/catch 吞掉，界面误报「连接中断」
- 主机卡内 sparkline 宽度塌缩到 30px
- PVE 客户机内存占满分配值被误报为红色（改为中性灰）
- 容器名断词错误（改为按连字符换行）

### 移除
- 乱码解码（scramble text）动效
