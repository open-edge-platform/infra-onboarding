# Summary of Changes to Upstream Open Source Project

This document summarizes the changes made to the upstream open source project [Tink Worker](https://github.com/tinkerbell/tink).
All the changes have been made on top of version `v0.10.0`.

## Overview

Forked repo is stored in the `tink-worker/` folder.
Changes have been made to enhance and customize the functionality for our specific use case.

## Changes

### Major Modifications

1. **cmd/tink-worker/worker/containerd.go**
    - New file to start / stop container using containerd.

2. **cmd/tink-worker/cmd/root.go**
    - Code change to remove docker client and use containerd.

3. **cmd/tink-worker/worker/container_manager.go**
    - Code change to remove docker client and use containerd.

4. **cmd/tink-worker/worker/log_capturer.go**
    - Code change to remove docker client and use containerd.

5. **cmd/tink-worker/worker/registry.go**
    - Code change to remove docker client and use containerd.

6. **go.mod**
    - Code change to remove docker client and use containerd.

7. **go.sum**
    - Code change to remove docker client and use containerd.

8. **internal/e2e**
    - Code removed since it's not used by tink worker.

9. **config**
    - Code removed since it's not used by tink worker.

10. **cmd/virtual-worker**
    - Code removed since it's not used by tink worker.

11. **cmd/tink-server**
    - Code removed since it's not used by tink worker.

12. **tink-worker/cmd/tink-controller**
    - Code removed since it's not used by tink worker.

13. **cmd/tink-agent**
    - Code removed since it's not used by tink worker.

14. **cmd/tink-worker/worker/containerd_test.go**
    - New file to fuzz test containerd code.

15. **VERSION**
    - New file to keep track of version.

16. **Makefile**
    - Updated file to enable CI.

17. **internal/server**
    - Code removed since it's not used by tink worker.

18. **api**
    - Code removed since it's not used by tink worker.

19. **internal/controller**
    - Code removed since it's not used by tink worker.

20. **internal/grpcserver**
    - Code removed since it's not used by tink worker.

21. **internal/hardware**
    - Code removed since it's not used by tink worker.

22. **internal/httpserver**
    - Code removed since it's not used by tink worker.

23. **internal/ptr**
    - Code removed since it's not used by tink worker.

24. **internal/testtime**
    - Code removed since it's not used by tink worker.

25. **internal/workflow**
    - Code removed since it's not used by tink worker.

### General Improvements

#### Linting

The following files have been added to skip some Trivy warnings.

- trivy.yaml
- .trivyignore

## Conclusion

These changes have been made to tailor the Tink Worker to use containerd directly,
and enhance the overall functionality. We will continue to maintain and update this fork as needed.

Last Updated Date: June 4, 2025
