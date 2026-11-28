# SelectOS LAN Device Agent

Small Go agent for Linux SelectOS endpoints. It enrolls once with the server's enrollment secret, stores a per-device credential with mode 0600, and sends a heartbeat every 30 seconds. It does not implement desktop login, remote command execution, reboot, or shutdown.

## Guided installation

From the root of a SelectOS repository checkout, install Go and run:

    sudo apt-get update
    sudo apt-get install -y golang-go
    sudo bash auth/agent/install-control-center-agent.sh

The installer asks for:
- The Control Center HTTPS URL.
- A unique device ID for this PC.
- The server enrollment token (input is hidden).

It builds the agent, creates the restricted service account and state directory, writes a root-only configuration file, installs the systemd unit, and starts the service. HTTP is refused by default; for an isolated lab network only, explicitly set SELECTOS_ALLOW_HTTP=1 when running the installer.

## Manual installation

Build from this directory:

    go build -trimpath -o selectos-agent .

Install the service account, binary and state directory:

    sudo useradd --system --home /var/lib/selectos-agent --shell /usr/sbin/nologin selectos-agent
    sudo install -d -o selectos-agent -g selectos-agent -m 0700 /var/lib/selectos-agent
    sudo install -m 0755 selectos-agent /usr/local/bin/selectos-agent

Create /etc/selectos-agent.env with mode 0600:

    SELECTOS_SERVER_URL=https://control-center.example.lan
    SELECTOS_DEVICE_ID=lab-pc-001
    SELECTOS_ENROLLMENT_TOKEN=REPLACE_WITH_SERVER_ENROLLMENT_SECRET
    SELECTOS_STATE_FILE=/var/lib/selectos-agent/credentials.json

Then install the unit and start it:

    sudo install -m 0644 selectos-agent.service /etc/systemd/system/selectos-agent.service
    sudo systemctl daemon-reload
    sudo systemctl enable --now selectos-agent
    sudo journalctl -u selectos-agent -f

## Behavior and security

Set the same random enrollment secret on the server as SELECTOS_DEVICE_ENROLLMENT_TOKEN. Use a unique ID per PC. The enrollment secret is needed only for initial registration; afterward the agent uses its unique device token. Keep the configuration and state file private. Never copy a device token to another PC.

The dashboard marks devices online within two minutes. Revoking a device invalidates its token; remove its local state and restore the enrollment secret before re-enrolling.

The agent is telemetry/heartbeat only. It does not execute commands received from the server. The PC remains usable without network access; the dashboard simply stops receiving fresh heartbeats. HTTPS with a trusted certificate is required for normal use.
