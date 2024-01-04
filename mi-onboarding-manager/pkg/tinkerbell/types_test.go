package tinkerbell

import (
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
