# Agent Notification

> 中文 | [English](README.en.md)

<p align="center">
  <img src="assets/agentnotify-promo.zh.png" alt="AgentNotify 中文宣传图：Claude Code、Codex 和 OpenClaw 任务通知" width="100%">
</p>

<p align="center">
  <strong>让 Agent 的任务状态及时出现在桌面通知里。</strong>
</p>

<p align="center">
  <img alt="Platform" src="https://img.shields.io/badge/platform-macOS%20%7C%20Windows-2f7cf6">
  <img alt="Language" src="https://img.shields.io/badge/language-%E4%B8%AD%E6%96%87%20%7C%20English-22a06b">
  <img alt="Discovery" src="https://img.shields.io/badge/discovery-mDNS%20%7C%20LAN-6b7280">
</p>

AgentNotify 是一个局域网桌面通知接收器。Claude Code、Codex、OpenClaw 等 Agent 可以把任务开始和结束事件发送给 AgentNotify，再由 macOS 或 Windows 显示系统原生通知。

桌面端负责运行本地通知服务、展示局域网访问地址、发送测试通知、设置开机启动和切换界面语言。Agent 侧配置由仓库内的 `agent-notify-discovery` skill 自动完成，通常不需要手写 hook 命令。

## 功能

| 能力 | 说明 |
|------|------|
| macOS / Windows 桌面端 | Tauri 应用内置 Go server sidecar，安装后直接运行 |
| 系统原生通知 | 使用 macOS 和 Windows 通知中心显示任务状态 |
| 局域网自动发现 | 通过 mDNS/DNS-SD 广播 `_agent-notify._tcp.local.` |
| 多 Agent 支持 | 支持 Claude Code、Codex、OpenClaw，可继续扩展 |
| 任务开始 / 结束通知 | 可按需配置 `start`、`stop`，或同时开启 |
| 本地控制面板 | 显示服务状态、局域网 URL、安装命令、历史记录和测试入口 |
| 开机启动 | 可在桌面端侧边栏开启或关闭随系统启动 |
| 中英文界面 | 应用设置中可切换中文或 English |

## 项目结构

| 路径 | 说明 |
|------|------|
| `tauri-app/` | macOS / Windows 桌面客户端 |
| `windows-server/` | Go 通知服务，桌面端打包为 sidecar |
| `skills/agent-notify-discovery/` | Agent 侧自动发现和 hook 配置 skill |
| `scripts/` | 安装、验证和辅助脚本 |
| `docs/` | 发布和维护文档 |

## 快速开始

### 1. 安装并启动桌面端

从 GitHub Release 下载对应平台的安装包。

macOS：

1. 打开 `.dmg` 文件。
2. 将 `AgentNotify` 拖入 `Applications`。
3. 启动 `AgentNotify`。
4. 首次弹出通知权限请求时点击 **Allow**。
5. 在应用里点击测试通知，确认系统通知可以正常显示。

Windows：

1. 运行 Windows 安装包。
2. 启动 `AgentNotify`。
3. 如果 Windows 防火墙或安全软件询问网络访问权限，允许局域网通信。
4. 在应用里点击测试通知，确认系统通知可以正常显示。

启动后，应用会显示本机局域网 URL，例如：

```text
http://192.168.1.23:17891
```

Agent 通常可以自动发现这个地址。如果 multicast 或跨网段环境导致自动发现失败，可以把应用里显示的 URL 交给配置 skill 手动指定。

### 2. 安装 AgentNotify Skill

在运行 Agent 的机器上，复制桌面端显示的安装命令，或直接安装本仓库的 skill：

```bash
npx skills add TaurusGGBOY/agent-notification
```

安装后，在 Agent 中运行：

```text
/agent-notify-discovery
```

常见配置目标：

```text
发现 AgentNotify，并为 Claude Code、Codex、OpenClaw 配置任务结束通知。
```

或者：

```text
使用 http://192.168.1.23:17891，为 Claude Code 和 Codex 配置 start、stop 通知。
```

配置完成后：

- 重启 Claude Code 或 Codex，让新的 hooks 生效。
- 重启 OpenClaw Gateway，让 Agent Notify plugin 生效。
- 如果 Codex 要求信任 hooks，先检查 `~/.codex/hooks.json`，确认内容后再批准。

### 3. 验证通知

重启 Agent 后发送一条简单消息：

```text
hi
```

任务结束时桌面端应该收到系统通知。如果配置了 `start`，任务开始时也会收到通知。

也可以用 HTTP 请求直接验证服务：

```bash
curl -X POST http://127.0.0.1:17891/notify \
  -H 'Content-Type: application/json' \
  -d '{"agent":"codex","event":"stop","project":"agent-notification","message":"manual test"}'
```

## 事件映射

| Agent hook | AgentNotify 事件 |
|------------|------------------|
| Claude Code `SessionStart` | `start` |
| Claude Code `Stop` | `stop` |
| Codex `SessionStart` | `start` |
| Codex `Stop` | `stop` |
| OpenClaw `before_model_resolve` | `start` |
| OpenClaw `agent_end` | `stop` |

## 通知协议

AgentNotify 接收 JSON 请求：

```http
POST /notify
```

```json
{
  "agent": "claude",
  "event": "start",
  "project": "agent-notification",
  "cwd": "/path/to/project",
  "message": "Task started",
  "timestamp": "2026-06-03T10:00:00Z"
}
```

字段说明：

| 字段 | 说明 |
|------|------|
| `agent` | Agent 名称，例如 `claude`、`codex`、`openclaw` |
| `event` | `start` 或 `stop` |
| `project` | 项目名 |
| `cwd` | Agent 运行目录 |
| `message` | 通知正文 |
| `timestamp` | 事件时间，建议使用 ISO 8601 |

## 排障

| 问题 | 处理方式 |
|------|----------|
| 桌面端没有通知 | 检查系统通知权限，并在 AgentNotify 中发送测试通知 |
| Agent 找不到桌面端 | 使用应用里显示的局域网 URL 手动配置 |
| 局域网机器无法访问 | 检查防火墙是否允许 `17891` 端口 |
| 修改 hooks 后不生效 | 重启对应 Agent 或 OpenClaw Gateway |
| Codex 提示 hook 信任 | 检查 `~/.codex/hooks.json` 后再批准 |

## 开发

桌面客户端位于 `tauri-app/`：

```bash
cd tauri-app
npm install
npm run tauri:dev
```

`npm run tauri:dev` 会为当前 Rust target 构建 Go sidecar，并启动 Tauri 桌面应用。

Go 通知服务位于 `windows-server/`：

```bash
cd windows-server
go test ./...
go run .
```

默认服务地址：

```text
0.0.0.0:17891
```

可通过环境变量覆盖：

```bash
AGENT_NOTIFY_HTTP_ADDR=127.0.0.1:17891 go run .
```

## 发布

发布流程见 [docs/release-checklist.md](docs/release-checklist.md)。发布需要人工执行；普通合并不自动代表发版。
