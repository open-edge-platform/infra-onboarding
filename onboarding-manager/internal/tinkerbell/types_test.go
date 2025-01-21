// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package tinkerbell

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
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

	wf := &Workflow{
		Version:       "0.1",
		Name:          "debian",
		GlobalTimeout: 1800,
		Tasks: []Task{{
			Name:       "os-installation",
			WorkerAddr: "{{.device_1}}",
			Volumes:    []string{"/dev:/dev", "/dev/console:/dev/console"},
			Actions: []Action{{
				Name:    "stream-image",
				Image:   "quay.io/tinkerbell-actions/image2disk:v1.0.0",
				Timeout: 600,
			}},
		}},
	}

	r, err := marshalWorkflow(wf)
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

func TestNewRebootTemplateData(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "TestNewRebootTemplateData_ValidName",
			args: args{
				name: "name",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewRebootTemplateData(tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewRebootTemplateData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewRebootTemplateData() = %v, want %v", got, tt.want)
			}
		})
	}
}
