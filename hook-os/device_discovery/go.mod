// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

module device-discovery

// remains with Go 1.24.1 till EMT Go is updated to support Go 1.24.4
go 1.24.1

require (
	github.com/open-edge-platform/infra-onboarding/onboarding-manager v1.33.0
	golang.org/x/oauth2 v0.28.0
	google.golang.org/grpc v1.73.0
)

require (
	cloud.google.com/go/compute/metadata v0.6.0 // indirect
	github.com/envoyproxy/protoc-gen-validate v1.2.1 // indirect
	golang.org/x/net v0.41.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.26.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250324211829-b45e905df463 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
)
