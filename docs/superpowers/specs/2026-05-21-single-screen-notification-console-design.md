# Single Screen Notification Console Design

## Goal

重构 AgentNotify Windows 客户端为一个更清晰的单屏控制台。界面去掉低价值搜索框和重复信息，保留通知服务最核心的控制：局域网地址、通知样式、样式预览、mDNS 广播开关、通知历史、测试与刷新。

## Scope

本次改动覆盖：

- Tauri 客户端 UI 结构与样式。
- 前端 API 类型与数据加载。
- Go server 的 `/manifest` 局域网地址。
- Go server 的通知历史接口。
- mDNS 广播开关的运行时控制。

本次不覆盖：

- 通知 PNG 生成逻辑重写。
- Windows toast 底层投递方式替换。
- 多页面路由。
- 长期持久化通知历史。历史只保存在 server 进程内存中，重启后清空。

## Layout

窗口保持 `1200x675`，单屏展示，不出现应用内部滚动条。

左侧只作为品牌与状态栏，不再显示导航项：

- Brand：AgentNotify / 通知控制台。
- 服务状态：在线 / 离线。
- 局域网地址：显示后端返回的 LAN URL，例如 `http://localhost:17891`。
- 左下角版本：显示客户端/server 版本，例如 `v1.0.0`。

顶部区域：

- 标题：通知控制台。
- 副标题：本机 Agent 通知服务、样式、广播和历史。
- 明暗模式按钮。
- 运行状态 pill。

主区域使用 2 列卡片，全部一屏可见：

1. 通知样式
2. 通知预览
3. 局域网广播
4. 通知历史
5. 快捷操作

## Removed UI

删除以下界面元素：

- “输入命令或搜索操作”输入框。
- 事件模式。
- 事件开关。
- “卡片”通知样式选项。
- 所有重复出现的样式显示，只在“通知样式”选择器中显示一次。
- “当前通知”卡片中重复展示通知样式的内容。

## Notification Styles

保留四种样式：

- `clean`：简洁
- `status-color`：状态
- `agent-badge`：徽章
- `compact`：紧凑

移除 `custom-card` 在客户端 UI 中的选择入口。后端可以继续兼容旧配置，但前端不会再主动展示或设置该样式。如果当前配置为 `custom-card`，前端加载时按 `clean` 显示并保存时改为 `clean`。

## Preview Differences

通知预览必须体现四种样式差异：

- 简洁：普通 toast，白底/深色卡片，标题 + 来源 + 内容，结构最少。
- 状态：左侧有明显状态色条，显示“启动”或“停止”状态标签。
- 徽章：突出 Agent 圆形徽章，项目名和 agent 名更醒目。
- 紧凑：一行或近似一行布局，突出时间、事件、项目，适合高频通知。

预览是前端模拟，不要求调用真实 Windows toast。

## LAN Address

后端 `/manifest` 的 `url` 不再直接使用请求的 `r.Host`，因为本机 UI 请求通常是 `127.0.0.1:17891`。

新增 LAN IP 检测逻辑：

- 枚举本机网络接口地址。
- 选择第一个非 loopback、非 link-local 的 IPv4。
- 优先返回局域网地址，例如 `192.168.x.x`、`10.x.x.x`、`172.16-31.x.x`。
- 如果找不到可用地址，回退到请求 host。

UI 只显示 `/manifest.url`。

## Broadcast Toggle

“快捷操作”里的重启按钮改为 `局域网广播` toggle。

语义：

- 开：启动 mDNS 广播，局域网设备可以发现 AgentNotify。
- 关：停止 mDNS 广播，HTTP 服务仍然继续运行，本机 UI 和直接访问 LAN URL 仍可用。

默认值：

- server 启动后默认开启广播。

API：

- `GET /broadcast` 返回 `{ "enabled": true | false }`
- `POST /broadcast` 请求 `{ "enabled": true | false }`
- `POST /broadcast` 生效后立即启动或停止 mDNS 广播。

前端加载状态时请求 `/broadcast`。切换 toggle 后调用 `POST /broadcast` 并刷新状态。

## Notification History

后端新增内存通知历史，记录最近 3 条有效 `/notify` 请求。

记录条件：

- JSON payload 合法。
- event 是 `start` 或 `stop`。
- 即使当前通知事件被配置过滤，仍记录历史，因为它反映请求到达过服务。

记录字段：

- `time`：server 接收时间，ISO 8601。
- `agent`
- `event`
- `project`
- `message`

API：

- `GET /history`
- 返回：

```json
{
  "items": [
    {
      "time": "2026-05-21T22:30:00+08:00",
      "agent": "tauri",
      "event": "start",
      "project": "AgentNotify",
      "message": "来自 AgentNotify 的测试通知"
    }
  ]
}
```

UI：

- 通知历史卡片默认显示最近 3 条。
- 每条展示时间、事件、项目/agent、摘要内容。
- 没有历史时显示空状态：“暂无通知记录”。
- 点击“测试”后应新增一条历史并刷新 UI。

## API Compatibility

现有接口继续保留：

- `/health`
- `/manifest`
- `/notify`
- `/settings`
- `/config`

新增接口：

- `/history`
- `/broadcast`

现有 `enabledEvents` 配置继续保留给后端使用，但 UI 不再暴露事件开关。

## Error Handling

前端：

- `/history` 或 `/broadcast` 请求失败时，不阻塞主界面。
- 历史失败显示“历史加载失败”。
- 广播状态失败显示“未知”，toggle 禁用。

后端：

- `/broadcast` 只接受 GET 和 POST。
- POST body 不是合法 JSON 时返回 400。
- `enabled` 字段缺失时返回 400。
- mDNS 启停失败时返回 500，并记录日志。

## Verification

本地验证：

- `cd tauri-app && npm run build`
- `cd windows-server && go test ./... -v`

Windows 验证：

- 同步前端和 Go server 文件到 Windows。
- `cd D:\project\agent-notification\windows-server; go test ./... -v`
- `cd D:\project\agent-notification\tauri-app; npm run tauri:build`
- 使用 `skills/windows-ui-screenshot/scripts/capture_windows_ui.py` 拉截图。

截图验收标准：

- 窗口完整显示，无应用内部滚动条。
- 搜索框不存在。
- 左侧没有“首页 / 通知 / 预览 / 设置”导航。
- 服务地址显示 LAN IP，不是 `127.0.0.1`。
- 版本在左下角。
- 样式只在“通知样式”模块出现一次。
- 没有“卡片”样式。
- 四种预览视觉明显不同。
- 没有事件开关。
- 有局域网广播 toggle，默认开启。
- 有通知历史模块，最多显示 3 条。
