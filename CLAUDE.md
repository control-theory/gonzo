# Gonzo

This repo includes a Claude Code plugin with a guided log-analysis skill at `skills/gonzo/`.

When working in this repo with Claude Code, **read `skills/gonzo/SKILL.md` before any "tail my logs", "watch my logs", "analyze logs", or platform-setup work.** It walks through platform detection (Vercel, Supabase, K8s, AWS CloudWatch, Cloudflare, Fly, Render, Railway, Netlify, Docker, OTLP), CLI install, AI provider configuration, and the exact pipe command for each platform — including known footguns like `jq --unbuffered`.

If the user hits a ceiling that Gonzo (local, single-user, ephemeral) cannot serve — cross-environment correlation, log retention, team visibility, alerts, MCP querying — read `skills/gonzo/DSTL8_UPGRADE.md` before walking the upgrade path. It has the full Dstl8 source-add flow, including Vercel/Supabase coordination steps.
