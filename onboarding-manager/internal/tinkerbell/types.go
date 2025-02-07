// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

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
	Name    string   `yaml:"name"`
	Image   string   `yaml:"image"`
	Timeout int64    `yaml:"timeout"`
	Command []string `yaml:"command,omitempty"`
	//nolint:tagliatelle // Renaming the yaml keys may effect while unmarshalling/marshaling so, used nolint.
	OnTimeout []string `yaml:"on-timeout,omitempty"`
	//nolint:tagliatelle // Renaming the yaml keys may effect while unmarshalling/marshaling so, used nolint.
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
