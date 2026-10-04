# DIodide Mac mini setup

The proxy runs on the Mac mini as `com.diodide.cliproxyapi`. The matching CPAMC
console lives in [DIodide/Cli-Proxy-API-Management-Center](https://github.com/DIodide/Cli-Proxy-API-Management-Center).
It defaults to dark mode and a quota ledger inspired by the reference screenshot.

## Open the console and connect four accounts

On the laptop:

```sh
~/.local/bin/cliproxy-console
```

This prepares the OAuth callback tunnel and opens the Tailscale console.
Devices signed into Tailscale as `DIodide@github` enter without a management key.
The proxy binds only to the Mac mini's loopback address; Tailscale and SSH provide
remote access. Keep both machines connected to Tailscale and the Mac mini awake.

The console is also available directly over Tailscale HTTPS, without an SSH tunnel:
[Open quota homepage](https://timmy-mac-mini.tail6d5626.ts.net:8443/management.html#/quota).
The Mac mini's existing Tailscale service on port 443
is preserved; this console uses port 8443 and remains tailnet-only.
The persistent Serve mapping is configured with:

```sh
ssh mac-mini 'tailscale serve --bg --https=8443 http://127.0.0.1:8317'
```

Passwordless access uses Tailscale Serve's verified identity headers, an explicit
login/origin allowlist, and same-origin browser checks. The server must remain
loopback-only. Other local processes are trusted, as in Tailscale's recommended
[Serve identity-header setup](https://tailscale.com/docs/features/tailscale-serve).
Other tailnet identities, shared external users, and tagged devices are not
automatically granted management access. Normal inference API keys remain required.

The installer reads `~/.config/cliproxyapi/tailnet-access.json` on the Mac mini:

```json
{"login":"DIodide@github","origin":"https://timmy-mac-mini.tail6d5626.ts.net:8443"}
```

It sets `MANAGEMENT_TAILSCALE_LOGIN` and `MANAGEMENT_TAILSCALE_ORIGIN` in launchd.
Removing that file and rerunning the installer disables passwordless access.
The management key remains available for recovery and the direct SSH-tunneled
URL <http://127.0.0.1:8317/management.html#/quota>, which still requires the key.

Codex rows show a dedicated **Manual resets** section with the banked count and
individual expiry dates/countdowns after refreshing quota. Zero available resets
and an unreported count are displayed distinctly. Reset actions require the
existing confirmation; viewing or refreshing quota does not consume a reset.

1. Open **OAuth Login**, choose Claude, and complete account one's login.
2. Repeat Claude login for account two using a separate browser profile or private
   window so the provider does not silently reuse account one.
3. Repeat for Codex accounts one and two.
4. Confirm **Auth Files** shows four separate credentials, then open **Quota
   Management** and click **Refresh all**.

The tunnel includes OAuth callback ports 1455 (Codex) and 54545 (Claude). If a
provider's browser redirect fails, use the console's callback-URL submission
field with the complete redirect URL from that same login attempt. Do not share
callback URLs or credential files. No upstream account was connected during installation.

The weekly summary sums the remaining percentages of loaded, general weekly
windows. Two fully available accounts display 200%. Unknown accounts are not
counted as zero, and model-specific limits remain in each account's row. These
percentages are not a token-normalized pool: different subscription tiers can
have different absolute capacities.

## Use the CLIs

On the laptop (launchers start/reuse the tunnel automatically):

```sh
~/.local/bin/cliproxy-models
~/.local/bin/claude-proxy
~/.local/bin/codex-proxy
```

Choose an exact model returned by `cliproxy-models` when needed:

```sh
claude-proxy --model MODEL_ID
codex-proxy --model MODEL_ID
```

Launchers live in `~/.local/bin`. If that directory is not on PATH, use the full
paths above. Ordinary `claude` and `codex` retain their existing login and settings.
The wrappers set credentials only for the launched process. The Claude launcher
sets `ANTHROPIC_BASE_URL` and `ANTHROPIC_AUTH_TOKEN`; the Codex launcher defines a
custom provider using the Responses API at `http://127.0.0.1:8317/v1`, with
`supports_websockets=true`. Codex OAuth files default to upstream WebSockets in
this fork, including newly connected and imported accounts. An explicit
`websockets: false` account setting is preserved. Normal Codex messages prefer
WebSockets on both hops; HTTP remains available for compact requests and upgrade
fallbacks. Deprecated `responses_websockets*` feature flags are not required.

On the Mac mini, both CLIs and launchers are installed:

```sh
ssh mac-mini
cd /path/to/your/project
claude-proxy
codex-proxy
```

Routing favors cache reuse with these defaults in the installer and example config:

```yaml
routing:
  strategy: "fill-first"
  session-affinity: true
  session-affinity-ttl: "24h"
  session-affinity-subagents: true
```

New sessions prefer the first available credential in each provider/model pool,
so the two accounts are not used evenly. Sessions remain on their bound account;
subagents inherit the parent's account when the client supplies a parent reference.
If that account becomes unavailable, automatic failover rebinds the session to
another available account. Recovery of the first account does not move an already
bound session back. Two credential attempts per retry round suit the two-account pool.

Native Codex session IDs and prompt cache keys, and Claude Code session identities,
are used for affinity. Keep using the same conversation when continuing work;
the proxy preserves native cache keys rather than imposing a shared key on all
conversations. The 24-hour setting retains account bindings since last use, not
upstream prompt caches. Bindings are in memory and reset on service restart or
selector replacement. Actual cache hits depend on the provider and matching prompt
content, and need verification after accounts are connected. Limits still belong
to the upstream accounts. Choose round-robin if balanced utilization or parallel
throughput later matters more than account concentration.

Keep Claude on Claude models and Codex on Codex models initially; cross-provider
protocol translation is available but needs separate testing.

## Desktop apps

### Codex

The macOS app reads the same `~/.codex/config.toml` as the CLI. Start the tunnel
first, then merge this provider into that file. Keep the two root settings before
any `[table]`; back up the file and preserve existing settings. Replace `MODEL_ID`
with an exact available model from `cliproxy-models`.

```toml
model_provider = "diodide-desktop"
model = "MODEL_ID"

[model_providers.diodide-desktop]
name = "DIodide Mac mini"
base_url = "http://127.0.0.1:8317/v1"
wire_api = "responses"
supports_websockets = true

[model_providers.diodide-desktop.auth]
command = "/bin/cat"
args = ["/Users/ibraheemamin/.config/cliproxyapi/client-key"]
timeout_ms = 5000
refresh_interval_ms = 300000
```

Restart the desktop app and create a new chat. The command-backed credential
avoids depending on shell environment variables that Dock-launched apps may not
inherit. Do not combine `auth` with `env_key` or `requires_openai_auth` for this
provider. The desktop example uses a separate provider ID from the CLI launcher to keep
their authentication mechanisms independent.
The desktop default was not changed during installation because the pool is empty.

Official guide: <https://learn.chatgpt.com/docs/enterprise/connect-to-a-gateway>.
Gateway connectivity does not automatically provide every native ChatGPT feature,
connector, cloud execution mode, or model capability.

### Claude Desktop

Current Claude documentation routes the desktop app through **third-party
inference configuration**, rather than shell `ANTHROPIC_BASE_URL` or CLI settings.
Open **Help → Troubleshooting → Enable Developer Mode**, then **Developer →
Configure Third-Party Inference**. Enter `http://127.0.0.1:8317` and the client key
from `~/.config/cliproxyapi/client-key` where the form requests gateway credentials.
Start the tunnel first. Availability and the exact form depend on the app version
and managed organization configuration. This integration has not been validated
with connected accounts. Gateway mode uses local sessions; SSH/cloud environments
and Remote Control are unavailable in that mode per the official guide.

Official guide: <https://code.claude.com/docs/en/llm-gateway-connect#desktop-app>.
For working directly on Mac mini files, SSH into it and use the CLI launchers.

## Service, credentials, and updates

On the Mac mini:

- Source: `~/Projects/CLIProxyAPI` and `~/Projects/Cli-Proxy-API-Management-Center`.
- Config: `~/.config/cliproxyapi/config.yaml`.
- Upstream OAuth files: `~/.config/cliproxyapi/auths/`.
- Client/management keys: `~/.config/cliproxyapi/client-key` and `management-key`.
- Console asset: `~/.config/cliproxyapi/static/management.html`.
- Logs: `~/.config/cliproxyapi/logs/`, with server log rotation capped at 100 MB.
- LaunchAgent: `~/Library/LaunchAgents/com.diodide.cliproxyapi.plist`.

Secrets are outside the Git checkouts with owner-only permissions. The laptop has
copies of the two proxy access keys; upstream OAuth credentials stay on the mini.
Management keys and client keys are different. Do not paste either into Git.

```sh
ssh mac-mini 'launchctl print gui/$(id -u)/com.diodide.cliproxyapi'
ssh mac-mini 'launchctl kickstart -k gui/$(id -u)/com.diodide.cliproxyapi'
# Stop until manually bootstrapped or the next user login:
ssh mac-mini 'launchctl bootout gui/$(id -u)/com.diodide.cliproxyapi'
# Close the laptop tunnel:
ssh -S ~/.config/cliproxyapi/ssh-tunnel -O exit mac-mini
```

The service starts at user login and restarts after process failures. A LaunchAgent
is not a pre-login system daemon. The laptop tunnel is restarted by the launchers
when needed. The console's automatic upstream updates are disabled to preserve the
custom build.

To update intentionally: review upstream changes, rebuild the proxy with Go 1.26+
and the console with `bun install --frozen-lockfile && bun run verify`. The Go
binary was built on the Apple Silicon laptop for the same macOS/arm64 target as
the mini. Copy both artifacts to the mini and run:

```sh
python3 deploy/macos/install-service.py /path/to/cli-proxy-api /path/to/management.html
```

The installer preserves existing config and keys. It replaces the executable and
console, and restarts the service. Back up config/auth files privately before
updates. Account inference, streaming, tools, quota fetches, and account failover
must be tested after the four OAuth logins; an empty pool cannot validate them.
