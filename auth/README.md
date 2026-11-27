# SelectOS authentication development

This directory contains the Go-based local authentication API for the Debian LAN server. The server is packaged with Docker Compose and PostgreSQL.

**This does not yet replace the current Gbox login UI.** The current boot image contains a precompiled legacy Gate initramfs; its source and build recipe are not in this repository. A client login application and initramfs integration are a separate required phase.

Start with [server/README.md](server/README.md) for Debian Docker Compose deployment. The API currently uses HTTP for isolated-LAN testing only; add TLS and firewall restrictions before production use.