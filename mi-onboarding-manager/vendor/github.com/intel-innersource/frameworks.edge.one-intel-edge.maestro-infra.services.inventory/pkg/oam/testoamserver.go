// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package oam

import (
	"os"
	"os/signal"
	"sync"
	"syscall"

	"google.golang.org/grpc/test/bufconn"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/util"
)

const (
	bufferSize = util.Megabyte
)

type TestOAM struct {
	readyChan chan bool
	servaddr  string
	termChan  chan bool
	wg        sync.WaitGroup
}

// testing related function.
func NewTestOAM() *TestOAM {
	toam := TestOAM{
		wg:        sync.WaitGroup{}, // waitgroup to wait for clean exit
		readyChan: make(chan bool),  // to signal readiness during the tests
		termChan:  make(chan bool),
		servaddr:  "bufconn",
	}

	return &toam
}

func (toam *TestOAM) StartTestOAM() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigChan
		close(toam.readyChan)
		close(toam.termChan)
	}()

	err := error(nil)

	toam.wg.Add(1) // Add grpc server

	// https://pkg.go.dev/google.golang.org/grpc/test/bufconn#Listener
	buffer := bufferSize
	blis := bufconn.Listen(buffer)
	// provide bufconn to client implementation during testing - server uses this too
	TestBufconn = blis
	TestReadyChan = toam.readyChan

	go func() {
		if err = StartOamGrpcServer(toam.termChan, toam.readyChan, &toam.wg, toam.servaddr, false); err != nil {
			zlog.Fatal().Err(err).Msg("Cannot start OAM gRPC server")
		}
	}()

	// create an unspecified test client
	TestClient = NewGrpcClient(
		toam.termChan,
		&toam.wg,
		toam.servaddr,
	)
}

func (toam *TestOAM) StopTestOAM() {
	// close unspecified test client
	TestClient.Close()
	// stop the server after tests
	close(toam.readyChan)
	close(toam.termChan)
	// wait until servers terminate
	toam.wg.Wait()
}
