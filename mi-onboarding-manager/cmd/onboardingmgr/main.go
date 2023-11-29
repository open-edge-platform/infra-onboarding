/*
   Copyright (C) 2023 Intel Corporation
   SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"
	"flag"

	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/intel-sandbox/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/pkg/logger"
	"github.com/intel-sandbox/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/pkg/maestro"
	"github.com/intel-sandbox/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/pkg/maestro/controller"
)

var (
	log        = logger.GetLogger()
	invSvrAddr = flag.String("invsvraddr", "localhost:50051", "Inventory server address to connect to")
)

func main() {
	flag.Parse()
	wg := &sync.WaitGroup{}
	ctx := context.Background()

	invClient, invEvents, err := maestro.NewInventoryClient(ctx, wg, *invSvrAddr)
	if err != nil {
		log.Fatalf("failed to create NewInventoryClient: %v", err)
	}

	nbHandler, err := controller.NewNBHandler(invClient, invEvents)
	if err != nil {
		log.Fatalf("failed to create NBHandler: %v", err)
	}

	err = nbHandler.Start()
	if err != nil {
		log.Fatalf("failed to start NBHandler: %v", err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigChan
		log.Info("Shutdown server")
		nbHandler.Stop()
		invClient.Close()
	}()

	wg.Wait()
}
