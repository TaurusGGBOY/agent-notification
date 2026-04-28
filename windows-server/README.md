# Agent Notify Windows Server

Run for development:

```powershell
go run .
```

Open settings:

```text
http://localhost:17891/settings
```

Test health:

```powershell
curl http://localhost:17891/health
```

Send test notification:

```powershell
curl -Method POST http://localhost:17891/notify `
  -ContentType "application/json" `
  -Body '{"agent":"claude","event":"stop","project":"agent-notification","message":"manual test"}'
```

LAN access:

```text
http://<windows-lan-ip>:17891/settings
```

Environment variables:

```text
AGENT_NOTIFY_HTTP_ADDR=0.0.0.0:17891
```