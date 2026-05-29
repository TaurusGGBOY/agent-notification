import {
  restartService,
  saveConfig,
  sendTestNotification,
  type AgentConfig,
  type EventName,
  type NotificationStyle,
} from "./api";

export interface CommandResult {
  message: string;
  config?: AgentConfig;
}

const styleAliases: Record<string, NotificationStyle> = {
  clean: "clean",
  status: "status-color",
  badge: "agent-badge",
  compact: "compact",
};

export async function runCommand(input: string, config: AgentConfig | null): Promise<CommandResult> {
  const command = input.trim().toLowerCase();
  if (!command) return { message: "请输入命令" };

  if (command === "test" || command === "send test") {
    await sendTestNotification("start");
    return { message: "已发送测试通知" };
  }

  if (command === "restart") {
    await restartService();
    return { message: "已重启服务" };
  }

  if (!config) return { message: "配置尚未加载" };

  if (command === "pause") {
    const next = { ...config, enabledEvents: [] as EventName[] };
    await saveConfig(next);
    return { message: "已暂停通知", config: next };
  }

  if (command === "resume") {
    const next = { ...config, enabledEvents: ["start", "stop"] as EventName[] };
    await saveConfig(next);
    return { message: "已恢复通知", config: next };
  }

  const eventMatch = command.match(/^(start|stop)\s+(on|off)$/);
  if (eventMatch) {
    const event = eventMatch[1] as EventName;
    const enabled = eventMatch[2] === "on";
    const events = new Set(config.enabledEvents);
    if (enabled) events.add(event);
    else events.delete(event);
    const next = { ...config, enabledEvents: [...events] as EventName[] };
    await saveConfig(next);
    return { message: `${event === "start" ? "启动" : "停止"}事件已${enabled ? "开启" : "关闭"}`, config: next };
  }

  const styleMatch = command.match(/^style\s+([a-z-]+)$/);
  if (styleMatch) {
    const style = styleAliases[styleMatch[1]];
    if (!style) return { message: `未知样式：${styleMatch[1]}` };
    const next = { ...config, notificationStyle: style };
    await saveConfig(next);
    return { message: `已切换样式为${labelForStyle(style)}`, config: next };
  }

  return { message: `未知命令：${input}` };
}

function labelForStyle(style: NotificationStyle): string {
  const labels: Record<NotificationStyle, string> = {
    clean: "简洁",
    "status-color": "状态",
    "agent-badge": "徽章",
    compact: "紧凑",
  };
  return labels[style];
}
