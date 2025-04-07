# Summary of Changes to Upstream Open Source Project

This document summarizes the changes made to the upstream open source project [Tinkerbell Hook](https://github.com/tinkerbell/hook).
All the changes have been made on top of version `v0.10.0`.

## Overview

Forked repo is stored in the `hook/` folder.
Changes have been made to enhance and customize the functionality for our specific use case.

## Changes

### Major Modifications

1. **files/dhcp.sh**
    - During startup, a one-shot DHCP call is executed on the interface that matches the `worker_id` (MAC Address)
provided in the kernel arguments, instead of running DHCP on all `eth*` interfaces.
    - This ensures that the IP address assignment is reliably completed before the `device-discovery` stage.

2. **images/hook-bootkit/main.go**
    - Added feature to measure and relay telemetry data on the time taken by the bootkit stage for KPI measurement.
    - Implemented `RestartPolicy` for tink-worker to ensure it restarts if the caddy endpoint is not yet available.

3. **images/hook-bootkit/go.mod**
    - Updated `github.com/docker/docker` module to address [CVE-2024-41110](https://nvd.nist.gov/vuln/detail/cve-2024-41110)

4. **images/hook-bootkit/go.sum**
    - Updated `github.com/docker/docker` module to address [CVE-2024-41110](https://nvd.nist.gov/vuln/detail/cve-2024-41110)

5. **images/hook-docker/Dockerfile**
   - Replaced `docker:26.1.0-dind` base image with custom `hook_dind` image.
   - This change addresses the security vulnerability related to port `2376` by keeping the port closed.

6. **images/hook-docker/main.go**
   - Replaced syslog port `514` with `5140`.
   - Added syslog format configuration `rfc3164`.
   - This change enables streaming data to fluent-bit.
   - Added 3 second delay post detecting reboot trigger file to provide sufficient time for tink-worker to send workflow
     execution success to tink controller

7. **kernel/configs/generic-6.6.y-x86_64**
   - Set flags to enable XZ Compression.

8. **kernel/configs/generic-5.10.y-x86_64**
   - Set flags to enable DM Verity.

## Conclusion

These changes have been made to tailor the Tinkerbell Actions to our specific requirements, improve performance,
and enhance the overall functionality. We will continue to maintain and update this fork as needed.

Last Updated Date: April 7, 2025
