# Pre-Merge CI Pipeline Checks

This document describes all the checks performed by the pre-merge CI pipeline
in the `.github/workflows/pre-merge.yml` workflow.

## Overview

The pre-merge pipeline runs automatically on pull requests to the `main` and
`release-*` branches. It performs comprehensive validation to ensure code
quality, security, and compliance before merging changes.

## Checks Performed

| Check Name | Description | Purpose | When It Runs |
|------------|-------------|---------|--------------|
| **Verify Branch Name** | Validates that the branch name follows the repository's naming conventions | Ensures consistent branch naming across the project | Always, for all PRs |
| **Discover Changed Subfolders** | Identifies which subfolders/components have been modified in the PR | Optimizes CI by only running checks on changed components | Always, for all PRs |
| **Filter Out Unwanted Changed Subfolders** | Filters out specific folders (`.github`, `.reuse`, `LICENSES`) from the changed projects list | Separates infrastructure changes from code changes | Always, for all PRs |
| **Markdown Lint (mdlint)** | Lints all markdown files using markdownlint-cli v0.44.0 | Ensures documentation follows markdown best practices and is consistently formatted | When root files or documentation changes |
| **License Check** | Validates SPDX license headers and compliance using the REUSE tool | Ensures all files have proper licensing information per Apache 2.0 requirements | When root files or license-related changes occur |
| **Version Check** | Validates version information in project files | Ensures version numbers are properly maintained and incremented | For changed subprojects |
| **Dependency Version Check** | Checks that dependency versions are up-to-date and compatible | Prevents using outdated or incompatible dependencies | For changed subprojects |
| **Build** | Compiles and builds the project code | Verifies that the code compiles without errors | For changed subprojects |
| **Lint** | Runs language-specific linters on the codebase | Enforces code style, identifies potential bugs, and maintains code quality | For changed subprojects |
| **Test** | Executes the test suite | Validates functionality and prevents regressions | For changed subprojects |
| **Validate Clean Folder** | Ensures no unwanted or generated files are committed | Keeps the repository clean and prevents accidental commits of build artifacts | For changed subprojects |
| **Docker Build** | Builds Docker container images for the components | Verifies that Docker images can be successfully built | For changed subprojects |
| **Trivy Security Scan** | Scans Docker images and configurations for security vulnerabilities | Identifies and prevents security issues before merging | For changed subprojects |
| **Final Status Check** | Aggregates results from all previous checks | Ensures all checks passed before allowing merge | Always, at the end of pipeline |

## Pipeline Stages

### Stage 1: Pre-Checks

This stage runs first and determines which subsequent checks need to run based
on the files that changed in the PR.

- Checkout code
- Verify branch name
- Discover changed subfolders
- Filter changes to determine scope

### Stage 2: Root-Level Checks

Runs when changes affect repository root files, GitHub workflows, or licensing:

- Markdown linting
- License compliance check

### Stage 3: Project-Specific Checks

Runs for each changed subproject (onboarding-manager, dkam, hook-os, etc.):

- Version validation
- Dependency version check
- Build verification
- Code linting
- Unit/integration tests
- Clean folder validation
- Docker image build
- Security vulnerability scanning

### Stage 4: Final Validation

Aggregates all check results to determine overall PR status.

## Running Checks Locally

You can run most of these checks locally before pushing:

```bash
# Run all checks
make all

# Run individual checks
make lint          # Run linting
make test          # Run tests
make build         # Build projects
make mdlint        # Lint markdown files
make license       # Check licensing
```

## Required Tools

- **markdownlint-cli**: v0.44.0
- **Python**: 3.13
- **Node.js**: 18
- **REUSE tool**: Installed via requirements.txt
- **Trivy**: Security scanner for containers

## Skipped Images in Trivy Scan

The following images are excluded from Trivy scanning:

- `postgres:16.4`
- `quay.io/tinkerbell/hook-containerd:30084036-amd64`
- `quay.io/tinkerbell/hook-runc:f0dbe53f-amd64`

## Additional Information

For more details about contributing and the development workflow, see:

- [Contributor's Guide](https://docs.openedgeplatform.intel.com/edge-manage-docs/main/developer_guide/contributor_guide/index.html)
- [Troubleshooting](https://docs.openedgeplatform.intel.com/edge-manage-docs/main/developer_guide/troubleshooting/index.html)
