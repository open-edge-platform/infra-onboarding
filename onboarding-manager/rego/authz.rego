# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

package authz

import future.keywords.in

# This query checks if caller has write access to the resource
hasWriteAccess {
    some role in input["realm_access/roles"] # iteration
    # We expect:
    # - with MT: [PROJECT_UUID]_en-agent-rw or [PROJECT_UUID]_en-ob
    regex.match("^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}_en-agent-rw$|^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}_en-ob$", role)
}

# This query checks if caller has read access to the resource
hasReadAccess {
    some role in input["realm_access/roles"] # iteration
    # We expect:
    # - with MT: [PROJECT_UUID]_en-agent-rw or [PROJECT_UUID]_en-ob
    regex.match("^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}_en-agent-rw$|^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}_en-ob$", role)
}
