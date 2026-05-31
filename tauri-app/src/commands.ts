import {
  restartService,
  saveConfig,
  sendTestNotification,
  type AgentConfig,
  type EventName,
} from "./api";

export interface CommandResult {
  message: string;
  config?: AgentConfig;
}

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

  return { message: `未知命令：${input}` };
}
