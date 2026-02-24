#!/bin/bash

# SPDX-FileCopyrightText: (C) 2026 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

set -Eeuo pipefail

APP_NAME="Modular vPro"
DRY_RUN="${DRY_RUN:-0}"

SCRIPT_DIR=$(pwd)
MANIFEST_CANDIDATES=(
    "${SCRIPT_DIR}/.success_install_status"
    "${SCRIPT_DIR}/install_pkgs_status"
    "${SCRIPT_DIR}/.base_pkg_install_done"
)

SERVICES=(
    "device-discovery-agent.service"
    "node-agent.service"
    "platform-manageability-agent.service"
)

FILES=(
    "/etc/apt/sources.list.d/edge-node.list"
    "/etc/apt/trusted.gpg.d/edge-node.asc"
    "/etc/apparmor.d/opt.edge-node.bin.platform-manageability-agent"
    "/etc/apparmor.d/opt.edge-node.bin.node-agent"
    "/etc/apparmor.d/opt.edge-node.bin.device-discovery-agent"
)

DIRS=(
    "/etc/edge-node"
    "/etc/intel_edge_node"
    "/var/lib/edge-node"
)

log() { printf '[%s] %s\n' "$(date '+%F %T')" "$*"; }
warn() { printf '[%s] WARN: %s\n' "$(date '+%F %T')" "$*" >&2; }

run() {
    if [[ "$DRY_RUN" == "1" ]]; then
        log "DRY_RUN: $*"
    else
        "$@"
    fi
}

require_root() {
    if [[ "${EUID:-$(id -u)}" -ne 0 ]]; then
        warn "Run as root: sudo $0"
        exit 1
    fi
}

remove_path() {
    local p="$1"
    [[ -z "$p" || "$p" == "/" ]] && return 0
    if [[ -L "$p" || -f "$p" || -d "$p" ]]; then
        log "Removing $p"
        run rm -rf -- "$p"
    fi
}

stop_disable_service() {
    local svc="$1"
    command -v systemctl >/dev/null 2>&1 || return 0

    if systemctl list-unit-files --type=service --no-legend 2>/dev/null | awk '{print $1}' | grep -Fxq "$svc"; then
        log "Stopping service $svc"
        run systemctl stop "$svc" || true
        log "Disabling service $svc"
        run systemctl disable "$svc" || true
    fi

    remove_path "/etc/systemd/system/${svc}"
    remove_path "/lib/systemd/system/${svc}"
    remove_path "/usr/lib/systemd/system/${svc}"
}

remove_manifest_entries() {
    local mf
    for mf in "${MANIFEST_CANDIDATES[@]}"; do
        [[ -f "$mf" ]] || continue
        log "Processing manifest: $mf"
        while IFS= read -r line; do
            line="${line#"${line%%[![:space:]]*}"}"
            line="${line%"${line##*[![:space:]]}"}"
            [[ -z "$line" || "${line:0:1}" == "#" ]] && continue
            remove_path "$line"
        done <"$mf"
        remove_path "$mf"
        return 0
    done
    return 1
}

main() {
    require_root
    trap 'warn "Uninstall failed at line $LINENO"' ERR

    log "Starting ${APP_NAME} uninstall"

    for svc in "${SERVICES[@]}"; do
        stop_disable_service "$svc"
    done

    run systemctl daemon-reload || true
    run systemctl reset-failed || true

    # Remove packages
    log "Removing packages"
    run apt-get remove -y node-agent platform-manageability-agent device-discovery-agent 2>/dev/null || true

    # Purge packages
    log "Purging packages"
    run apt-get purge -y node-agent platform-manageability-agent device-discovery-agent 2>/dev/null || true

    if ! remove_manifest_entries; then
        warn "No install manifest found. Falling back to default cleanup list."
    fi

    for f in "${FILES[@]}"; do
        remove_path "$f"
    done

    for d in "${DIRS[@]}"; do
        remove_path "$d"
    done

    # Remove agent user accounts and groups
    log "Removing agent user accounts and groups"
    run userdel pm-agent 2>/dev/null || true
    run userdel hd-agent 2>/dev/null || true
    run userdel device-discovery-agent 2>/dev/null || true
    run userdel node-agent 2>/dev/null || true
    run groupdel bm-agents 2>/dev/null || true

    # Remove sudoers files
    log "Removing sudoers files"
    remove_path "/etc/sudoers.d/pm-agent"
    remove_path "/etc/sudoers.d/hd-agent"
    remove_path "/etc/sudoers.d/device-discovery-agent"
    remove_path "/etc/sudoers.d/node-agent"

    log "${APP_NAME} uninstall completed"
}

main "$@"