# Summary of Changes to Upstream Open Source Project

This document summarizes the changes made to the upstream open source project [Tinkerbell Actions](https://github.com/tinkerbell/actions).

## Overview

Some tinker actions are forked from the open source [Tinkerbell Actions](https://github.com/tinkerbell/actions) repository.
Changes have been made to enhance and customize the functionality for our specific use case.

## Changes

### Modifications to Existing Actions

1. **image2disk**
   - Enhanced the logic to automatically detect the target disk based on size and type.
   - Improved error handling and logging for better troubleshooting.
   - Enabled SHA checksum validation for the source image.
   - Updated base build image to `golang:1.23.2-alpine3.20`. Updated final image to `alpine:3.20.3` to pass trivy scan.
   - Used `nsenter` in `CMD_LINE` to call the binary for security considerations

2. **cexec**
   - Enhanced the logic to automatically detect the target disk based on size and type.
   - Improved error handling and logging for better troubleshooting.
   - Updated base build image to `golang:1.23.2-alpine3.20`. Updated final image to `alpine:3.20.3` to pass trivy scan.
   - Used `nsenter` in `CMD_LINE` to call the binary for security considerations

3. **writefile**
   - Enhanced the logic to automatically detect the target disk based on size and type.
   - Improved error handling and logging for better troubleshooting.
   - Updated base build image to `golang:1.23.2-alpine3.20`. Updated final image to `alpine:3.20.3` to pass trivy scan.
   - Used `nsenter` in `CMD_LINE` to call the binary for security considerations

### General Improvements

- **Linting and Validation**
  - Added `shellcheck` and `yamllint` targets to the Makefile for linting shell scripts and YAML files.
  - Ensured that linting commands only run if there are files to lint.

- **Documentation Updates**
  - Updated the README.md file to include new actions and their descriptions.
  - Reformatted the documentation to ensure line lengths are less than 120 characters.

## Conclusion

These changes have been made to tailor the Tinkerbell Actions to our specific requirements, improve performance,
and enhance the overall functionality. We will continue to maintain and update this fork as needed.

Last Updated Date: February 18, 2025
