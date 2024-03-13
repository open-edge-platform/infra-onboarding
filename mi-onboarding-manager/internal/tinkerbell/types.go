// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package tinkerbell

import (
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type Workflow struct {
	Version       string `yaml:"version"`
	Name          string `yaml:"name"`
	ID            string `yaml:"id,omitempty"`
	GlobalTimeout int    `yaml:"global_timeout"`
	Tasks         []Task `yaml:"tasks"`
}

type Task struct {
	Name        string            `yaml:"name"`
	WorkerAddr  string            `yaml:"worker"`
	Actions     []Action          `yaml:"actions"`
	Volumes     []string          `yaml:"volumes,omitempty"`
	Environment map[string]string `yaml:"environment,omitempty"`
}

type Action struct {
	Name        string            `yaml:"name"`
	Image       string            `yaml:"image"`
	Timeout     int64             `yaml:"timeout"`
	Command     []string          `yaml:"command,omitempty"`
	OnTimeout   []string          `yaml:"on-timeout,omitempty"`
	OnFailure   []string          `yaml:"on-failure,omitempty"`
	Volumes     []string          `yaml:"volumes,omitempty"`
	Environment map[string]string `yaml:"environment,omitempty"`
	Pid         string            `yaml:"pid,omitempty"`
}

func marshalWorkflow(wf *Workflow) ([]byte, error) {
	return yaml.Marshal(wf)
}

func unmarshalWorkflow(yamlContent []byte) (*Workflow, error) {
	var workflow Workflow

	if err := yaml.Unmarshal(yamlContent, &workflow); err != nil {
		return &Workflow{}, errors.Wrap(err, "parsing yaml data")
	}
	return &workflow, nil
}

func NewTemplateData(name, ip, clientyp, disk, serial, tinkerversion string) ([]byte, error) {
	wf := Workflow{
		Version:       "0.1",
		Name:          name,
		GlobalTimeout: 8000,
		Tasks: []Task{{
			Name:       "os-installation-di",
			WorkerAddr: "{{.device_1}}",
			Volumes: []string{
				"/dev:/dev",
				"/dev/console:/dev/console",
				"/lib/firmware:/lib/firmware:ro",
			},
			Actions: []Action{
				{
					Name:    ActionStoringAlpine,
					Image:   "localhost:7443/one-intel-edge/edge-node/tinker-actions/store_alpine:" + tinkerversion,
					Timeout: 500,
					Environment: map[string]string{
						"BLOCK_DEVICE": disk,
						"PARTITION_SZ": "500MB",
					},
				},
				{
					Name:    ActionRunFDO,
					Image:   "localhost:7443/one-intel-edge/edge-node/tinker-actions/fdoclient_action:" + tinkerversion,
					Timeout: 400,
					Environment: map[string]string{
						"DATA_PARTITION_LBL": "CREDS",
						"FDO_RUN_TYPE":       "di",
						"FDO_MFGIP":          ip,
						"FDO_MPORT":          "8081",
						"DEVICE_SERIAL":      serial,
						"TYPE":               clientyp,
						"FDO_TLS":            "https",
					},
				},
				{
					Name:    ActionReboot,
					Image:   "public.ecr.aws/l0g8r8j6/tinkerbell/hub/reboot-action:latest",
					Timeout: 90,
					Volumes: []string{
						"/worker:/worker",
					},
				},
			},
		}},
	}

	return marshalWorkflow(&wf)
}
