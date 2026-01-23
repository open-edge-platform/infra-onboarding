// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

module device-discovery

// remains with Go 1.24.1 till EMT Go is updated to support Go 1.24.4
go 1.24.1

require (
	github.com/open-edge-platform/infra-onboarding/onboarding-manager v1.33.0
	golang.org/x/oauth2 v0.34.0
	google.golang.org/grpc v1.80.0-dev
)

require (
	cloud.google.com/go/compute/metadata v0.9.0 // indirect
	github.com/envoyproxy/protoc-gen-validate v1.3.0 // indirect
	golang.org/x/net v0.48.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/text v0.32.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251202230838-ff82c1b0f217 // indirect
	google.golang.org/protobuf v1.36.10 // indirect
)
