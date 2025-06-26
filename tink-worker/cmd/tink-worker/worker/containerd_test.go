// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package worker

import (
	"fmt"
	"strings"
	"testing"
)

func FuzzNewContainerName(f *testing.F) {
	f.Add("my-container")
	f.Add("container-123")
	f.Add(randStr(100))
	f.Fuzz(func(t *testing.T, containerName string) {
		name := newContainerName(containerName)
		if len(name) == 0 {
			t.Errorf("newContainerName() returned an empty string for input: %s", containerName)
		} else if len(name) > 76 {
			t.Errorf("newContainerName() returned a string longer than 76 characters: %s", name)
		}
	})
}

func FuzzParseCmdLine(f *testing.F) {
	registry := "docker.io"
	wid := "123"
	wimg := "my-worker-image"
	proxyHTTP := "http://proxy.example.com:8080"
	proxyHTTPS := "https://proxy.example.com:8443"
	f.Add(fmt.Sprintf("docker_registry=%s worker_id=%s tink_worker_image=%s HTTP_PROXY=%s HTTPS_PROXY=%s",
		registry, wid, wimg, proxyHTTP, proxyHTTPS))
	f.Fuzz(func(_ *testing.T, cmdLineStr string) {
		if len(cmdLineStr) == 0 {
			return
		}
		cmdLines := strings.Split(cmdLineStr, " ")
		if len(cmdLines) == 0 {
			return
		}
		_ = parseCmdLine(cmdLines)
	})
}
