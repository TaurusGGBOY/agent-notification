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
  card: "custom-card",
  custom: "custom-card",
};

export async function runCommand(input: string, config: AgentConfig | null): Promise<CommandResult> {
  const command = input.trim().toLowerCase();
  if (!command) return { message: "Type a command" };

  if (command === "test" || command === "send test") {
    await sendTestNotification("start");
    return { message: "Sent test notification" };
  }

  if (!config) return { message: "Config is not loaded yet" };

  if (command === "restart") {
    await restartService();
    return { message: "Restarted service" };
  }

  if (command === "pause") {
    const next = { ...config, enabledEvents: [] as EventName[] };
    await saveConfig(next);
    return { message: "Paused notifications", config: next };
  }

  if (command === "resume") {
    const next = { ...config, enabledEvents: ["start", "stop"] as EventName[] };
    await saveConfig(next);
    return { message: "Resumed notifications", config: next };
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
    return { message: `${event} events ${enabled ? "enabled" : "disabled"}`, config: next };
  }

  const styleMatch = command.match(/^style\s+([a-z-]+)$/);
  if (styleMatch) {
    const style = styleAliases[styleMatch[1]];
    if (!style) return { message: `Unknown style: ${styleMatch[1]}` };
    const next = { ...config, notificationStyle: style };
    await saveConfig(next);
    return { message: `Switched style to ${style}`, config: next };
  }

  return { message: `Unknown command: ${input}` };
}