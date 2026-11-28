# SelectOS LAN Device Agent

Small Go agent for Linux SelectOS endpoints. It enrolls once with the server's enrollment secret, stores a per-device credential with mode 0600, and sends a heartbeat every 30 seconds. It does not implement desktop login, remote command execution, reboot, or shutdown.

## Build

    cd auth/agent
    go build -trimpath -o selectos-agent .
    sudo install -m 0755 selectos-agent /usr/local/bin/selectos-agent
    sudo install -d -o selectos-agent -g selectos-agent -m 0700 /var/lib/selectos-agent

## Configure

Create /etc/selectos-agent.env (root-readable only):

    SELECTOS_SERVER_URL=http://10.0.0.10:8088
    SELECTOS_DEVICE_ID=lab-pc-001
    SELECTOS_ENROLLMENT_TOKEN=REPLACE_WITH_SERVER_ENROLLMENT_SECRET
    SELECTOS_STATE_FILE=/var/lib/selectos-agent/credentials.json

Set the same random enrollment secret on the server as SELECTOS_DEVICE_ENROLLMENT_TOKEN. Use a unique ID per PC. The enrollment secret is needed only for initial registration; afterward the agent uses its unique device token. HTTP is for isolated LAN testing only; production should use HTTPS with a trusted certificate.

Install the systemd unit and run:

    sudo chmod 600 /etc/selectos-agent.env
    sudo systemctl daemon-reload
    sudo systemctl enable --now selectos-agent
    sudo journalctl -u selectos-agent -f

The dashboard marks devices online within two minutes. Revoking a device invalidates its token; re-enrollment is required to restore reporting.
