# Cloud image maintenance scripts

This directory holds `prepare.sh`, `cleanup.sh`, `firstboot.sh` and their systemd units,
used to bake a cloud image template from a clean Linux instance. For an ordinary
installation, follow the getting-started section of the top-level
[`README.md`](../../README.md) instead.

`cleanup.sh` wipes the build host's data, secrets, SSH authorisations and caches and then
shuts the machine down, so it must only be run on a dedicated build host. The scripts
themselves are the authoritative reference for their arguments and behaviour.
