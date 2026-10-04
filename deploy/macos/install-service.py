#!/usr/bin/env python3
"""Install an already-built CLIProxyAPI binary and panel as a macOS LaunchAgent."""
import os
import json
from pathlib import Path
import plistlib
import secrets
import shutil
import subprocess
import sys
import time

binary, panel = map(Path, sys.argv[1:3])
home = Path.home()
state = home / '.config/cliproxyapi'
state.mkdir(parents=True, exist_ok=True, mode=0o700)
os.chmod(state, 0o700)
for directory in ['auths', 'static', 'logs', 'bin']:
    (state / directory).mkdir(exist_ok=True, mode=0o700)
config = state / 'config.yaml'
if not config.exists():
    api_key = secrets.token_urlsafe(32)
    management_key = secrets.token_urlsafe(32)
    for name, value in [('client-key', api_key), ('management-key', management_key)]:
        path = state / name
        path.write_text(value + '\n')
        path.chmod(0o600)
    config.write_text(f'''config-version: 8
server:
  host: "127.0.0.1"
  port: 8317
management:
  allow-remote: false
  secret-key: "{management_key}"
  disable-control-panel: false
  disable-auto-update-panel: true
  panel-github-repository: "https://github.com/DIodide/Cli-Proxy-API-Management-Center"
access:
  api-keys:
    - "{api_key}"
oauth:
  auth-dir: "{state / 'auths'}"
routing:
  strategy: "fill-first"
  session-affinity: true
  session-affinity-ttl: "24h"
  session-affinity-subagents: true
  retry:
    request-retry: 2
    max-retry-credentials: 2
    max-retry-interval: 15
observability:
  logs:
    debug: false
    logging-to-file: true
    logs-max-total-size-mb: 100
    error-logs-max-files: 5
    request-log: false
''')
    config.chmod(0o600)
label = 'com.diodide.cliproxyapi'
environment = {'MANAGEMENT_STATIC_PATH': str(state / 'static'), 'HOME': str(home)}
tailnet_access = state / 'tailnet-access.json'
if tailnet_access.exists():
    access = json.loads(tailnet_access.read_text())
    environment['MANAGEMENT_TAILSCALE_LOGIN'] = access['login']
    environment['MANAGEMENT_TAILSCALE_ORIGIN'] = access['origin']
plist = home / 'Library/LaunchAgents' / (label + '.plist')
plist.parent.mkdir(parents=True, exist_ok=True)
domain = f'gui/{os.getuid()}'
# Stage artifacts before stopping the running service; failed copies leave it intact.
replacements = []
for source, destination in [(binary, state / 'bin/cli-proxy-api'), (panel, state / 'static/management.html')]:
    if source.resolve() != destination.resolve():
        staged = destination.with_name(destination.name + '.next')
        shutil.copy2(source, staged)
        replacements.append((staged, destination))
subprocess.run(['launchctl', 'bootout', domain + '/' + label], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
for staged, destination in replacements:
    os.replace(staged, destination)
(state / 'bin/cli-proxy-api').chmod(0o700)
with plist.open('wb') as stream:
    plistlib.dump({
        'Label': label,
        'ProgramArguments': [str(state / 'bin/cli-proxy-api'), '--config', str(config)],
        'WorkingDirectory': str(state),
        'EnvironmentVariables': environment,
        'RunAtLoad': True,
        'KeepAlive': True,
        'ThrottleInterval': 10,
        'StandardOutPath': str(state / 'logs/launchd.out.log'),
        'StandardErrorPath': str(state / 'logs/launchd.err.log'),
    }, stream)
# launchd may finish unregistering a booted-out service asynchronously.
for attempt in range(6):
    result = subprocess.run(['launchctl', 'bootstrap', domain, str(plist)], capture_output=True, text=True)
    if result.returncode == 0:
        break
    if attempt == 5:
        raise RuntimeError(result.stderr.strip() or 'launchctl bootstrap failed')
    time.sleep(1)

print(f'Installed {label}; config and keys are in {state}.')
