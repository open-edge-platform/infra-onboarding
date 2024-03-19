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

func TestNewTemplateData(t *testing.T) {
	type args struct {
		name          string
		ip            string
		clientyp      string
		disk          string
		serial        string
		tinkerVersion string
	}
	wf := Workflow{}
	want, _ := marshalWorkflow(&wf)
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "Test Case 1",
			args: args{
				name:     "TestWorkflow",
				ip:       "000.0.0.0",
				clientyp: "testClient",
				disk:     "/dev/sda",
				serial:   "12345",
			},
			want:    want,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewDITemplateData(tt.args.name, tt.args.ip, tt.args.clientyp, tt.args.disk, tt.args.serial, tt.args.tinkerVersion)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDITemplateData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewDITemplateData() = %v, want %v", got, tt.want)
			}
		})
	}
}
