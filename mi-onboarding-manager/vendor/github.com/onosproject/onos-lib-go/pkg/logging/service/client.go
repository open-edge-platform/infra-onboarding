// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"fmt"

	"google.golang.org/grpc"
)

// Connect establishes a client-side connection to the gRPC end-point.
func Connect(address string, opts ...grpc.DialOption) *grpc.ClientConn {
	conn, err := grpc.Dial(address, opts...)
	if err != nil {
		fmt.Println("Can't connect", err)
	}
	return conn
}
