# 🌱 SubSage

> Take control of every subscription. Self-hosted, lightweight, AI Agent native.

SubSage is a self-hosted subscription tracker that works **with your AI Agent**. Through simple commands, let your AI Agent manage all your subscriptions. Copy one line, and your Agent handles the rest.

[English](./README.md) · [中文](./README_zh.md)

---

## 🤖 Deploy with Your AI Agent

Have a coding Agent like Cursor, Cline, or Windsurf? Copy this and send it:

> Clone https://github.com/wangjc683/subsage and deploy it with Docker Compose. After it starts, open http://localhost:8321 in my browser.

That's it. Your Agent handles `git clone`, `docker compose up -d`, and opens the app. Zero manual commands.

Want public access? Add a reverse proxy (Nginx or Caddy) with HTTPS.

> **Prerequisite**: Docker is required. Don't have it? Ask your Agent to install it for you.

---

## 🎬 Agent in Action

Once SubSage is running and connected to your Agent, managing subscriptions becomes a conversation:

**Track a new subscription**
> 🗣 *"I just subscribed to Claude Pro, $20/month, paying with Visa"*
>
> 🤖 *Done! Added Claude Pro — $20/mo, AI category. Your total monthly AI spending is now $45.*

**Check upcoming renewals**
> 🗣 *"What's renewing this week?"*
>
> 🤖 *2 renewals coming up: Spotify ($9.99, tomorrow) and iCloud+ ($2.99, Friday).*

**Manage subscriptions by voice**
> 🗣 *"Cancel Netflix and pause my Midjourney"*
>
> 🤖 *Done. Netflix → cancelled, Midjourney → paused. This saves you $28.98/month.*

**Spending insights**
> 🗣 *"How much am I spending on AI tools?"*
>
> 🤖 *5 AI subscriptions totaling $89/month. Top spenders: ChatGPT Plus ($20), Claude Pro ($20), Cursor ($20).*

**Smart actions**
> 🗣 *"I got the GitHub Student Pack, mark my Copilot as free"*
>
> 🤖 *Updated Copilot: price $0, discount note "GitHub Student Pack". Your dev spending dropped by $10/month.*

No app switching. No form filling. Just tell your Agent what happened.

If your Agent supports vision, you don't even need to type — just show it a screenshot of your bill, and your Agent + SubSage will handle the rest.

---

## ✨ Features

- 🤖 **AI Agent Native** — One-click install command, works with any Agent
- 💳 **Subscription Management** — Full CRUD, 10 built-in categories + custom
- 💱 **Multi-Currency** — Auto exchange rates, unified to your base currency
- 📊 **Spending Analytics** — Monthly/yearly/daily breakdown with charts
- 📅 **Calendar View** — Visual renewal timeline
- 💰 **Discount Tracking** — Original price, actual price, discount notes
- 📤 **Import/Export** — Excel and JSON backup
- 🌙 **Dark / Light / System** — Three theme modes
- 🌐 **i18n** — English & Chinese, auto-detected
- 📱 **Responsive** — Desktop + mobile, PWA installable
- 🔒 **Self-Hosted** — Your data stays on your machine

---

## 🚀 Quick Start

### Docker (Recommended)

```bash
git clone https://github.com/wangjc683/subsage.git
cd subsage
docker compose up -d
```

Open [http://localhost:8321](http://localhost:8321) — first visit will prompt you to create an admin account.

### Manual Setup

```bash
# Backend
cd backend && go run main.go

# Frontend (new terminal)
cd frontend && npm install && npm run dev
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `SAGE_DB_PATH` | SQLite database path | `/data/sage.db` |
| `SAGE_PORT` | Listen port | `8321` |
| `SAGE_JWT_SECRET` | JWT signing key (auto-generated if empty) | Auto |
| `TZ` | Timezone | `UTC` |

### Backup & Restore

```bash
docker cp subsage:/data/sage.db ./backup.db    # Backup
docker cp ./backup.db subsage:/data/sage.db     # Restore
docker restart subsage
```

---

## 🤖 Connecting Your Agent

1. Open SubSage → **Agent** in the sidebar
2. Click **"📋 Copy Install Command"**
3. Send it to your AI Agent
4. Your Agent installs the SubSage Skill automatically

All Agent API routes use `X-API-Token` header auth. See the [Agent page](http://localhost:8321/#/agent) for full API docs.

---

## 🛠 Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go 1.25 + Echo + SQLite |
| Frontend | Svelte 4 + Vite |
| Auth | bcrypt + JWT (Web) / API Token (Agent) |
| Charts | Chart.js |
| Exchange Rates | open.er-api.com (free, cached daily) |
| Deploy | Docker multi-stage build |

---

## 📄 License

[MIT](./LICENSE) — free and open source.
