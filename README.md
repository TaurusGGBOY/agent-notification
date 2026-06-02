# Agent Notification

<p align="center">
  <img src="assets/agentnotify-promo.png" alt="AgentNotify 宣传图：支持 Claude Code、Codex、OpenClaw 的任务开始和结束通知" width="100%">
</p>

<p align="center">
  <strong>让 Agent 的每一次任务，都及时触达。</strong>
</p>

<p align="center">
  <img alt="Platform" src="https://img.shields.io/badge/platform-macOS%20%7C%20Windows-2f7cf6">
  <img alt="Language" src="https://img.shields.io/badge/language-%E9%BB%98%E8%AE%A4%E4%B8%AD%E6%96%87%20%7C%20i18n-22a06b">
  <img alt="Discovery" src="https://img.shields.io/badge/discovery-mDNS%20%7C%20LAN-6b7280">
</p>

AgentNotify 是一个局域网通知接收器。Claude Code、Codex、OpenClaw 等 Agent 可以把任务开始和结束事件发送到 AgentNotify，由 macOS 或 Windows 桌面显示系统原生通知。

AgentNotify 支持多语言界面，默认语言为中文。配置过程由 AgentNotify skill 自动处理，通常不需要手动编写脚本命令。

## 特性

| 能力 | 说明 |
|------|------|
| 局域网自动发现 | 通过 mDNS/DNS-SD 广播 `_agent-notify._tcp.local.`，agent 可自动找到桌面端 |
| 原生桌面通知 | 支持 macOS 与 Windows 的系统通知中心 |
| 多 Agent 支持 | 支持 Claude Code、Codex、OpenClaw，并可扩展更多 agent |
| 开始 / 结束事件 | 可配置任务开始通知、任务结束通知，或两者同时开启 |
| 本地网络通信 | 默认面向局域网使用，服务监听 `0.0.0.0:17891` |
| 多语言界面 | 默认中文，可在应用设置中切换其他语言 |

## 组件

**桌面应用**（`tauri-app/`）

- 提供 macOS 和 Windows 一键安装包
- 打开 AgentNotify 界面，并启动内置 Go server sidecar
- 通过 `127.0.0.1:17891` 在本机控制服务
- 在 `0.0.0.0:17891` 接收局域网 agent 通知

**Agent 配置 Skill**（`skills/agent-notify-discovery/`）

- 自动发现局域网中的 AgentNotify
- 配置 Claude Code hooks
- 配置 Codex hooks
- 配置 OpenClaw 的 Agent Notify plugin
- 支持 `start` 和 `stop` 通知事件

## 快速开始

### 1. 安装并启动桌面端

#### macOS

1. 双击 `.dmg` 文件。
2. 将 `AgentNotify` 拖入 `Applications`。
3. 打开 `Applications`，启动 `AgentNotify`。
4. macOS 请求通知权限时，点击 **Allow**。如果没有出现提示，打开 **System Settings** -> **Notifications** -> **AgentNotify**，手动开启通知。
5. 在 AgentNotify 中点击 **Test**，确认右上角可以看到通知横幅。

#### Windows

1. 运行 Windows 安装包。
2. 启动 `AgentNotify`。
3. 如果 Windows 或安全软件询问网络访问权限，允许 AgentNotify 在局域网内通信。
4. 在 AgentNotify 中点击 **Test**，确认右下角可以看到通知横幅。

启动后，桌面应用会展示本机局域网 URL。agent 通常可以自动发现它；如果发现失败，把这个 URL 发给 agent 继续配置。

### 2. 安装 AgentNotify skill

在运行 agent 的机器上，复制 AgentNotify 界面中显示的 skill 安装命令，或安装本仓库的 skill：

```bash
npx skills add TaurusGGBOY/agent-notification
```

### 3. 让 agent 自动配置通知

在 agent 中运行已安装的 skill：

```text
/agent-notify-discovery
```

告诉 skill 发现 AgentNotify 并配置通知 hooks。Agent 会自行执行需要的配置步骤，不需要你手动输入 Python 命令。

可配置的通知时机：

- `stop`：任务结束时通知，适合只关心结果的场景
- `start` + `stop`：任务开始和结束都通知，适合需要感知 agent 工作状态的场景

配置完成后：

- 重启 Claude Code 或 Codex，让新的 hooks 生效
- 重启 OpenClaw Gateway，让 Agent Notify plugin 生效
- 如果 Codex 询问是否信任 hooks，请先检查 `~/.codex/hooks.json`，确认后再批准

### 4. 验证通知

重新启动 agent，发送一条简单消息：

```text
hi
```

当 agent 任务结束时，桌面应该出现通知。如果配置了 `start` 事件，任务开始时也会出现通知。

## 通知样式

| 样式 | 说明 |
|------|------|
| `clean` | 简洁、低干扰 |
| `status-color` | 按状态显示颜色，开始为蓝色，结束为绿色 |
| `agent-badge` | 显示 agent 标识 |
| `compact` | 更紧凑的通知布局 |

## 事件映射

| Agent hook | 通知事件 |
|------------|----------|
| `SessionStart` | `start` |
| `Stop` | `stop` |
| OpenClaw `before_model_resolve` | `start` |
| OpenClaw `agent_end` | `stop` |

## 通知协议

```http
POST /notify
```

```json
{
  "agent": "claude",
  "event": "start|stop",
  "project": "...",
  "cwd": "...",
  "message": "...",
  "timestamp": "..."
}
```

## 开发

桌面客户端位于 `tauri-app/`。它会打包 Go server sidecar，提供系统托盘和通知客户端界面。

```bash
cd tauri-app
npm install
npm run tauri:dev
```

Windows server 构建目标位于 `windows-server/`，需要 Go 1.21+。
