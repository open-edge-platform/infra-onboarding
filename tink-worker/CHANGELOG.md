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




### General Improvements

#### Linting

The following files have been added to skip some Trivy warnings.

- .trivyignore

## Conclusion

These changes have been made to tailor the Tink Worker to use containerd directly,
and enhance the overall functionality. We will continue to maintain and update this fork as needed.

Last Updated Date: June 4, 2025
