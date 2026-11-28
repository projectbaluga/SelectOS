# Connect a SelectOS Linux PC to Control Center

## Architecture and offline behavior

The Control Center is an optional management plane. SelectOS must boot and remain usable without the server, LAN, DNS, or internet. The native initramfs does not contact the Control Center and does not contain server credentials.

The Linux device agent runs **after the installed Linux operating system has booted**, under systemd. It enrolls the PC once and then reports heartbeats every 30 seconds. When the network is unavailable, the PC continues to work; heartbeat failures are logged and retried. The dashboard may show the device offline until connectivity returns.

This first integration reports device presence and basic metrics. It does not provide remote shell, remote desktop, reboot/shutdown, or centralized workstation login.

## Server prerequisites

1. Deploy the SelectOS server and Control Center using the server guide.
2. Configure a trusted TLS endpoint and use its HTTPS URL for normal LAN operation.
3. Set a long, unique SELECTOS_DEVICE_ENROLLMENT_TOKEN on the server.
4. Allow the PC to reach the API endpoint through the LAN firewall/reverse proxy. Do not expose PostgreSQL.

Do not use plain HTTP on an untrusted network: enrollment secrets and bearer tokens would be exposed. HTTP is suitable only for an isolated lab network.

## Install on a SelectOS Linux PC

Build the agent from the repository on the target Linux PC:

    sudo apt-get update
    sudo apt-get install -y golang-go git
    git clone https://github.com/projectbaluga/SelectOS.git
    cd SelectOS/auth/agent
    go build -trimpath -o selectos-agent .

Create a dedicated service account and state directory:

    sudo useradd --system --home /var/lib/selectos-agent --shell /usr/sbin/nologin selectos-agent
    sudo install -d -o selectos-agent -g selectos-agent -m 0700 /var/lib/selectos-agent
    sudo install -m 0755 selectos-agent /usr/local/bin/selectos-agent

Create the configuration without putting the enrollment secret in shell history:

    sudo install -m 0600 /dev/null /etc/selectos-agent.env
    sudoedit /etc/selectos-agent.env

Use these values in the file:

    SELECTOS_SERVER_URL=https://YOUR-CONTROL-CENTER-HOST
    SELECTOS_DEVICE_ID=unique-name-for-this-PC
    SELECTOS_ENROLLMENT_TOKEN=the-same-secret-configured-on-server
    SELECTOS_STATE_FILE=/var/lib/selectos-agent/credentials.json

Install and enable the service:

    sudo install -m 0644 selectos-agent.service /etc/systemd/system/selectos-agent.service
    sudo systemctl daemon-reload
    sudo systemctl enable --now selectos-agent
    sudo systemctl status selectos-agent
    sudo journalctl -u selectos-agent -f

The agent enrolls automatically on first successful connection. It stores its unique device token in the root-restricted state directory. After enrollment, the shared enrollment secret is not needed by the agent; it can be removed from /etc/selectos-agent.env and the service restarted. The device token remains in the state file.

## Verify in Control Center

Open the administrator dashboard and check the Devices inventory. The PC should appear after enrollment and become online after a heartbeat. Heartbeats are sent every 30 seconds; the dashboard uses a two-minute online window.

If it does not appear, check:
- systemctl status selectos-agent
- journalctl -u selectos-agent -n 100 --no-pager
- the HTTPS URL and certificate trust on the PC
- the enrollment secret matches the server
- the device ID is unique
- the server firewall/reverse proxy allows the API routes

## Security and operational notes

- Use a different device ID for every PC.
- Never copy /var/lib/selectos-agent/credentials.json to another PC.
- Keep /etc/selectos-agent.env and the agent state directory restricted.
- Revoking a device in Control Center invalidates its credential; remove its old local state and re-enroll to restore reporting.
- The agent is telemetry/heartbeat only. It does not execute commands received from the server.
- Offline PC use is unaffected by server outages. The dashboard simply stops receiving fresh heartbeats.
- This is currently a Linux/systemd integration. Windows and macOS agents are not included.
