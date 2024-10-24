# SPDX-FileCopyrightText: (C) 2024 Intel Corporation
# SPDX-License-Identifier: LicenseRef-Intel

package authz

import future.keywords.in

# This query checks if caller has write access to the resource
hasWriteAccess {
    some role in input["realm_access/roles"] # iteration
    # We expect:
    # - with MT: [PROJECT_UUID]_node-agent-readwrite-role or [PROJECT_UUID]_edge-onboarding-role
    # - without MT: node-agent-readwrite-role
    regex.match("^(([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}_)?node-agent-readwrite-role)|([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}_edge-onboarding-role)$", role)
}

# This query checks if caller has read access to the resource
hasReadAccess {
    some role in input["realm_access/roles"] # iteration
    # We expect:
    # - with MT: [PROJECT_UUID]_node-agent-readwrite-role or [PROJECT_UUID]_edge-onboarding-role
    # - without MT: node-agent-readwrite-role
    regex.match("^(([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}_)?node-agent-readwrite-role)|([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}_edge-onboarding-role)$", role)
}
