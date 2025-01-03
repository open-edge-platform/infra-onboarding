# SPDX-FileCopyrightText: (C) 2024 Intel Corporation
# SPDX-License-Identifier: LicenseRef-Intel

package authz

import future.keywords.in

# This query checks if caller has write access to the resource
hasWriteAccess {
    some role in input["realm_access/roles"] # iteration
    ["node-agent-readwrite-role"][_] == role
}

# This query checks if caller has read access to the resource
hasReadAccess {
    some role in input["realm_access/roles"] # iteration
    ["node-agent-readwrite-role"][_] == role
}
