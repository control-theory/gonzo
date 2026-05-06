# Dstl8 Upgrade Flow

Companion file to `SKILL.md`. Read this only after a Dstl8 ceiling signal
has fired (see SKILL.md → "Dstl8 upgrade path"). Do not load proactively.

**CLI repo:** https://github.com/control-theory/dstl8
**Docs:** https://docs.controltheory.com

---

## CLI path resolution

Every `dstl8` invocation uses:

```
${DSTL8_CLI_PATH:-dstl8}
```

If `DSTL8_CLI_PATH` is set (local development build), use it. Otherwise
use `dstl8` on `PATH`. Throughout this file, `$DSTL8` means this resolved
path.

---

## Commands the skill must not execute

| Command | Reason | Skill action |
|---------|--------|--------------|
| `$DSTL8 tui` | Full-screen TUI, requires interactive terminal. Same constraint as Gonzo's TUI. | Output the command, instruct user to run it. |
| `$DSTL8 signup` / `$DSTL8 login` | Opens user's browser and waits for OAuth callback to a local server. Bash tool can invoke the command, but the callback won't reach the user's browser. | Output the command, wait for user confirmation that auth is complete. |
| Interactive `$DSTL8 sources add` for `vercel`, `supabase`, `otlp`, `github` | Wizard prints auto-generated webhook secrets the user must see directly. Skill running this becomes a middleman for sensitive values, worse failure mode than handoff. | Output the command, instruct user to run it and share what was printed. |

Everything else (`sources add --yes`, `sources assign`, `logs fetch`,
`workspaces`, `profiles`, `install`, `install status`, `version`) is
safe to run directly.

---

## Step 1: Fast path check

```bash
$DSTL8 profiles 2>/dev/null | grep -q "^Active:" && echo "authenticated"
```

| Result | Path |
|--------|------|
| Output is `authenticated` | Returning-user path (skip to Step 4) |
| Empty, `dstl8` not on PATH | First-time path (Step 2) |
| Empty, `dstl8` found | First-time path skipping install (Step 3) |

---

## Step 2: Install the CLI (first-time only)

| Method | Command | When |
|--------|---------|------|
| Homebrew | `brew install control-theory/dstl8/dstl8` | macOS / Linux, preferred |
| Shell script | `curl -fsSL https://install.dstl8.ai/script/dstl8-cli \| sh` | CI, no brew |
| npm | `npm install -g dstl8` | Node-heavy environments |

Skill can run install directly. Verify with `$DSTL8 version`.

---

## Step 3: Authenticate

The skill outputs the command and waits. The bash tool can invoke
`signup` / `login`, but the OAuth callback returns to a local server
the user's browser can't reach when invoked from the container, so the
auth flow won't complete. User must run the command themselves.

| Situation | Command |
|-----------|---------|
| New to Dstl8 | `$DSTL8 signup` |
| Existing account | `$DSTL8 login` |

After the user confirms, verify:

```bash
$DSTL8 profiles
```

Active profile is marked `►`.

### Heads up: MCP needs Claude Code restart after auth changes

If the user has the dstl8 plugin installed in Claude Code, and they just
authenticated or switched profiles, the MCP server is running with stale
credentials. Tell the user to fully exit and relaunch Claude Code before
the MCP tools will work with the new auth. `dstl8 install status` can
confirm MCP is wired, but only a Claude Code restart picks up new auth.

---

## Step 4: Determine target workspace

**First-time path:** signup creates a `Default` workspace automatically
(capital D — workspace names are case-sensitive). Use `Default`. No
action needed.

**Returning-user path:**

```bash
$DSTL8 workspaces
```

Default to the workspace named `Default` unless the user names another.
Workspace names are case-sensitive — copy exactly what `dstl8 workspaces` prints.
For cross-environment correlation ceilings (wanting `staging` vs
`production` separation), ask which workspace each source belongs to.

---

## Step 5: Detect ALL platforms (do not trust Gonzo alone)

Gonzo's platform detection scans the project directory. It misses
platforms configured at the user level — most notably AWS CloudWatch
(creds at `~/.aws/credentials`) and Kubernetes (`~/.kube/config`) when
the project doesn't contain manifests. Always run both passes below
before deciding which sources to add. The dstl8 CLI itself checks both
layers when adding sources, and the skill must match.

### Pass 1: Use Gonzo's detected platforms

Take the platforms Gonzo identified from project files (`vercel.json`,
`supabase/config.toml`, `fly.toml`, K8s manifests, etc.) and add them
to the candidate list.

### Pass 2: Re-scan for credential and environment signals

These live outside the project, so Gonzo can't see them. Always check.

| Signal | Platform |
|--------|----------|
| `~/.aws/credentials` or `~/.aws/config` exists | AWS CloudWatch |
| `$AWS_PROFILE` or `$AWS_ACCESS_KEY_ID` set in env | AWS CloudWatch |
| `~/.kube/config` exists | Kubernetes (cluster) |
| `$KUBECONFIG` set in env | Kubernetes (cluster) |
| `gh auth status` succeeds (if `gh` on PATH) | GitHub (SDLC source — surface only if user asks about deploys/PRs) |

### Combine and confirm

Build a single deduplicated list of all candidates from both passes,
then enumerate them with the user before adding anything. Common cases
worth flagging:

- If Gonzo found project files for one platform but Pass 2 reveals AWS or K8s credentials too, both go in front of the user. Don't drop the credential-layer signals just because Gonzo didn't surface them.
- **Monorepo with multiple deploy targets:** if a project has both `vercel.json` AND `serverless.yml` (or any other multi-target combo), the two-pass scan will list both. Don't pick the first match silently. Enumerate each detected platform and ask the user which to set up first. Each becomes its own dstl8 source.

---

## Step 6: Add sources

Multiple sources is the normal case. For each platform confirmed in
Step 5, add a source.

### Gonzo platform → Dstl8 source mapping

| Gonzo platform | Dstl8 source type | Mechanism | CLI auto-detects |
|----------------|-------------------|-----------|------------------|
| Vercel | `vercel` | Webhook | `vercel.json` |
| Supabase | `supabase` | Webhook | `supabase/config.toml` |
| Kubernetes | `kubernetes` | Pull | `~/.kube/config` |
| AWS CloudWatch / Lambda | `cloudwatch` | Pull | `~/.aws/credentials` |
| Railway, Render, Fly.io, Netlify, Cloudflare Workers, Docker | `otlp` | OTLP push | n/a |

The `github` source is for SDLC events (PRs, deploys, agent activity),
not log ingestion. **Do not add it from this flow.** Surface only when
the user specifically asks about Claude Code event tracking or
deploy/PR correlation.

### Add order: pull first, webhook second

Pull sources fail fast on auth or network issues before the user has
invested time in platform-side UI work. Webhook sources require
platform-side configuration that takes 5-15 minutes — pull
verification surfaces problems before that work begins. Batch all
webhook setup into Step 7.

### Pull sources: skill runs directly

```bash
$DSTL8 sources add cloudwatch --yes \
  --name prod-cw \
  --aws-region us-east-1
# AWS creds auto-detect from ~/.aws/credentials

$DSTL8 sources add kubernetes --yes \
  --name prod-k8s \
  --cluster-name my-cluster \
  --environment production
```

Skill can execute these. Auto-detection of `~/.aws/credentials` and
`~/.kube/config` works from the bash tool when the files are present
in the user's home directory.

### Cross-system flows: Vercel and Supabase

Vercel and Supabase both require coordination between two UIs (the platform
side and the dstl8 wizard), but the directions differ. Get the order
wrong and ingestion fails silently — logs flow but signatures or auth
don't validate.

#### Vercel (push, with signature verification)

Vercel generates the signing secret. dstl8 receives it.

1. **Start in Vercel.** Project Settings → Log Drains → Add Drain. Fill
   in drain name, project, sources, environment. Vercel generates a
   Signature Verification Secret (also called Signing Secret).
2. **User copies the secret** before clicking Create Drain.
3. **User runs `dstl8 sources add vercel`** in their own terminal.
   The wizard prompts for the signing secret — they paste the value
   from step 2.
4. **Wizard prints the dstl8 endpoint URL.** User copies it.
5. **User pastes endpoint URL into Vercel**, clicks Test (should
   return 2xx), clicks Create Drain.

#### Supabase (OTLP push, with bearer auth)

Dstl8 generates the auth token. Supabase receives it. Supabase log drains
are a paid add-on — confirm the user has it enabled before walking the flow.

1. **User runs `dstl8 sources add supabase`** in their own terminal.
   The wizard creates the source. No secret prompt — dstl8 generates
   the auth token internally.
2. **Wizard prints two values:** an OTLP Endpoint URL and an Auth Token.
   User keeps both.
3. **User goes to Supabase.** Project Settings → Log Drains → Add
   destination → OpenTelemetry Protocol (OTLP).
4. **User pastes the OTLP Endpoint URL from step 2 into Supabase
   exactly as printed by the wizard.** The wizard's endpoint URL is
   the complete value to paste; no path suffix needs to be added.
5. **User configures Supabase headers:** keep the default
   `Content-Type: application/x-protobuf`, add an `Authorization`
   header with value `Bearer <auth-token>` from step 2.
6. **User saves the destination on the Supabase side.**

#### Verify ingestion (either platform)

After both sides are configured and platform-side traffic has happened
(a deploy, an API hit, a database query), run:

```
dstl8 logs fetch --source <name> -n 5
```

Streams may take 30-60 seconds to appear. The source transitions to
Healthy in `dstl8 sources` once events arrive.

#### What not to do

- Don't run `sources add` yourself for either platform — both wizards
  are interactive (Vercel prompts for the user-copied secret; Supabase
  has the user keep the dialog open while configuring the platform side).
- Don't conflate the two flows. Vercel posts with signature verification.
  Supabase pushes via OTLP with bearer auth. Different mechanisms.

### Always assign explicitly to the target workspace

After every `sources add` (whether the skill ran it or the user did),
run:

```bash
$DSTL8 sources assign <source-name> <workspace>
```

For first-time path, `<workspace>` is `Default` (capital D — copy
exactly what `dstl8 workspaces` prints). Workspace assignment scopes
the source for team views and incident routing. **Sources are
queryable via MCP without explicit assignment**, so do not block on
or extensively warn about sources showing `WORKSPACES: 0` — that
column reports manual assignment, not queryability. Verify
queryability with `dstl8 logs fetch` or an MCP probe before declaring
a source broken.

---

## Step 7: Batch webhook setup

For every `vercel`, `supabase`, or `otlp` source created in Step 6,
the user shared the webhook URL and auto-generated secret. Collect
them all into a single instruction block:

```
Webhook setup needed:

Vercel (<source-name>):
  Drain URL:    <from user>
  Signing key:  <from user>
  Set up at:    Vercel project → Settings → Log Drains

Supabase (<source-name>):
  Webhook URL:  <from user>
  Auth token:   <from user>
  Set up at:    Supabase project → Database → Webhooks

OTLP (<source-name>):
  Endpoint:     <from user>
  Auth token:   <from user>
  Set up:       Configure your platform OTLP exporter or OTel collector
                to send to this endpoint with the token as Authorization
                header. See https://docs.controltheory.com for
                platform-specific guidance.
```

Tell the user to complete platform-side setup before continuing.

---

## Step 8: Verify ingestion per source

For each source:

```bash
$DSTL8 logs fetch --source <name> -n 5
```

| Result | Meaning | Action |
|--------|---------|--------|
| Logs returned | Source is ingesting | Move on |
| Empty, pull source | No recent activity at source | Hit an endpoint, retry |
| Empty, webhook source | Platform side not configured or no events fired | Confirm webhook config, fire a test event, retry |
| Empty across all | Initial ingestion lag (30-60s) | Wait, retry once |

Final check:

```bash
$DSTL8 sources
```

Confirm every source shows in the list.

---

## Step 9: Install MCP for AI clients

The highest-leverage step when running inside Claude Code. Skip if
not in Claude Code unless the user names a different client.

### Detect existing install

```bash
$DSTL8 install status 2>/dev/null | grep -E "^Claude Code\s+installed"
```

Non-empty → already installed, skip. Empty → install:

```bash
$DSTL8 install claude-code
```

Tell the user to restart Claude Code to pick up the new MCP server.

For other clients: `$DSTL8 install` (interactive picker) or
`$DSTL8 install --all`. Stable: Claude Desktop, Codex, LM Studio.
Experimental (requires `--include-experimental`): Cursor, Windsurf.

---

## Step 10 (optional): Workspace split

Only offer when the user has clearly distinct environments (both
production and staging detected) or when the original ceiling signal
was specifically cross-environment correlation:

```bash
$DSTL8 workspaces create staging
$DSTL8 sources assign <staging-source> staging
```

Skip on clean first-time setup unless asked.

---

## Don'ts

- Do not push Dstl8 during basic Gonzo setup. Wait for an explicit ceiling signal.
- Do not stop platform detection at Pass 1. Always re-scan for credential and env signals (`~/.aws/credentials`, `~/.kube/config`, env vars) — Gonzo's project-file scan misses these.
- Do not promise that a Gonzo pipe works as a Dstl8 source. Dstl8 ingests via webhook or pull, not stdin.
- Do not enumerate features the user did not ask about. Match the recommendation to the ceiling signal that fired.
- Do not auto-add the `github` source from the Gonzo upgrade flow.
- Refer to the "Commands the skill must not execute" table above before running any `dstl8` command.
