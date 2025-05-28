// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package tinkerbell_test

import (
	"gopkg.in/yaml.v2"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/tinkerbell"
)

func TestMarshal(t *testing.T) {
	wfStr := `
    version: "0.1"
    name: debian
    global_timeout: 1800
    tasks:
    - name: "os-installation"
      worker: "{{.device_1}}"
      actions:
      - name: "stream-image"
        image: "quay.io/tinkerbell-actions/image2disk:v1.0.0"
        timeout: 600
      volumes:
      - /dev:/dev
      - /dev/console:/dev/console
`

	wfExpected, err := unmarshalWorkflow([]byte(wfStr))
	if err != nil {
		t.Errorf(`Got unexpected error: %v"`, err)
	}

	wf := &tinkerbell.Workflow{
		Version:       "0.1",
		Name:          "debian",
		GlobalTimeout: 1800,
		Tasks: []tinkerbell.Task{{
			Name:       "os-installation",
			WorkerAddr: "{{.device_1}}",
			Volumes:    []string{"/dev:/dev", "/dev/console:/dev/console"},
			Actions: []tinkerbell.Action{{
				Name:    "stream-image",
				Image:   "quay.io/tinkerbell-actions/image2disk:v1.0.0",
				Timeout: 600,
			}},
		}},
	}

	r, err := yaml.Marshal(wf)
	if err != nil {
		t.Errorf(`Got unexpected error: %v"`, err)
	}
	wfGot, err := unmarshalWorkflow(r)
	if err != nil {
		t.Errorf(`Got unexpected error: %v"`, err)
	}

	if !assert.EqualValues(t, wfExpected, wfGot) {
		t.Errorf(`Got unexpected result: got "%v" wanted "%v"`, wfGot, wfExpected)
	}
}
