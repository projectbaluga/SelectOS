# SelectOS authentication development

This directory introduces the Go-based local authentication API for the Debian LAN server.

**This does not yet replace the current Gbox login UI.** The current boot image contains a precompiled legacy Gate initramfs; its source and build recipe are not in this repository. A client login application and initramfs integration are a separate required phase. The API is independently buildable and testable now.

Start with [server/README.md](server/README.md). Do not deploy the initial HTTP listener outside a trusted LAN; add TLS and firewall restrictions before production use.
