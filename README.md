# 🌱 SubSage

> Take control of every subscription. Self-hosted, lightweight, AI Agent native.

SubSage is a self-hosted subscription tracker that works **with your AI Agent**. Through simple commands, let your AI Agent manage all your subscriptions. Copy one line, and your Agent handles the rest.

[English](./README.md) · [中文](./README_zh.md)

---

## 🤖 Deploy with Your AI Agent

Have a coding Agent like Cursor, Cline, or Windsurf? Copy one of these and send it:

**Option A — Script install (no Docker needed):**

> Run `curl -fsSL https://raw.githubusercontent.com/wangjc683/subsage/main/install.sh | bash` to install SubSage. After it starts, open http://localhost:8321 in my browser.

**Option B — Docker:**

> Run `docker run -d --name subsage -p 8321:8321 -v subsage-data:/data wangjc683/subsage`. After it starts, open http://localhost:8321 in my browser.

That's it. Your Agent runs one command and opens the app. Zero manual setup.

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

## 📸 Screenshots

| Overview | Subscriptions |
|----------|---------------|
| ![Overview](docs/screenshots/overview.png) | ![Subscriptions](docs/screenshots/subscriptions.png) |

| Calendar | Edit Modal |
|----------|------------|
| ![Calendar](docs/screenshots/calendar.png) | ![Edit Modal](docs/screenshots/edit-modal.png) |

| Agent Setup | |
|-------------|---|
| ![Agent](docs/screenshots/agent.png) | |

### 🌙 Dark Mode

| Overview | Subscriptions |
|----------|---------------|
| ![Overview Dark](docs/screenshots/overview-dark.png) | ![Subscriptions Dark](docs/screenshots/subscriptions-dark.png) |

### 📱 Mobile

| Overview | Subscriptions |
|----------|---------------|
| ![Overview Mobile](docs/screenshots/overview-mobile.png) | ![Subscriptions Mobile](docs/screenshots/subscriptions-mobile.png) |

---

## 🚀 Quick Start

### One-Line Install (no Docker needed)

```bash
curl -fsSL https://raw.githubusercontent.com/wangjc683/subsage/main/install.sh | bash
```

Auto-detects your OS (Linux/macOS) and architecture (amd64/arm64), downloads the binary, and registers a system service.

### Docker (Recommended for servers)

**One command:**

```bash
docker run -d --name subsage -p 8321:8321 -v subsage-data:/data wangjc683/subsage
```

**Or with Docker Compose:**

```bash
curl -O https://raw.githubusercontent.com/wangjc683/subsage/main/docker-compose.yml
docker compose up -d
```

Open [http://localhost:8321](http://localhost:8321) — first visit will prompt you to create an admin account.

> Supports `linux/amd64` and `linux/arm64` (NAS, Raspberry Pi, Apple Silicon).

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

## 🔄 Upgrade

SubSage stores all data in a SQLite database. Upgrading is safe — your data is always preserved.

### Script Install Upgrade

```bash
# Re-run the install script — it downloads the latest version automatically
curl -fsSL https://raw.githubusercontent.com/wangjc683/subsage/main/install.sh | bash
```

### Docker Upgrade

```bash
# 1. Backup (recommended)
docker cp subsage:/data/sage.db ./sage-backup-$(date +%Y%m%d).db

# 2. Pull latest image & restart
docker pull wangjc683/subsage:latest
docker stop subsage && docker rm subsage
docker run -d --name subsage -p 8321:8321 -v subsage-data:/data wangjc683/subsage
```

### Agent-Assisted Upgrade

SubSage is Agent-native — even upgrades can be a conversation. Send this to your Agent:

> Check the current version of my SubSage instance, then check the latest release at https://github.com/wangjc683/subsage/releases. If there's a newer version, show me the changelog and ask if I want to upgrade. If I confirm, run the upgrade and verify the new version.

### Version History

See the full changelog at [GitHub Releases](https://github.com/wangjc683/subsage/releases).

| Version | Highlights |
|---------|-----------|
| v0.3.0 | Calendar heatmap visualization, cross-page category drill-down, comprehensive UI/UX audit (11 fixes: hover-reveal actions, unified pill styles, design token migration, trend line polish) |
| v0.2.3 | Delete bug fix, full i18n audit (zero hardcoded strings), page transitions, calendar category colors, graduated overdue badges, keyboard shortcuts, dark mode polish |
| v0.2.2 | List/grid view toggle, modal form redesign (segmented controls, styled selects), overview layout merge, dynamic daily cost hints, a11y cleanup |
| v0.2.1 | Calendar redesign (inline details, stats bar, smart navigation), Settings theme dropdown fix |
| v0.2.0 | Mobile bottom tab bar, theme/logout in Settings, Notion-style dark mode, breathing micro-interactions |
| v0.1.1 | Bug fixes: calendar refresh, theme label, category names; Mobile UX polish |
| v0.1.0 | Shared edit modal, Calendar UX overhaul, Chart.js fix, i18n polish |
| v0.0.1 | Initial release — full CRUD, Agent API, multi-currency, i18n |

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
