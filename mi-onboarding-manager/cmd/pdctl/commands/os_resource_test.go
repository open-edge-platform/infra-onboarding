/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/
package commands

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOsCmds(t *testing.T) {
	actual := new(bytes.Buffer)
	RootCmd := OsCmds()
	RootCmd.SetOut(actual)
	RootCmd.SetErr(actual)
	RootCmd.SetArgs([]string{"create", "--addr=localhost:50053", "--insecure", "--profileName=name"})
	err := RootCmd.Execute()
	assert.Error(t, err)
}

func TestOsCmds_GetById(t *testing.T) {
	actual := new(bytes.Buffer)
	RootCmd := OsCmds()
	RootCmd.SetOut(actual)
	RootCmd.SetErr(actual)
	RootCmd.SetArgs([]string{"getById", "--addr=localhost:50054", "--insecure", "--resource-id=123"})
	err := RootCmd.Execute()
	assert.Error(t, err)
}

func TestOsCmds_Get(t *testing.T) {
	actual := new(bytes.Buffer)
	RootCmd := OsCmds()
	RootCmd.SetOut(actual)
	RootCmd.SetErr(actual)
	RootCmd.SetArgs([]string{"get", "--addr=localhost:50055", "--insecure"})
	err := RootCmd.Execute()
	assert.Error(t, err)
}

func TestOsCmds_Delete(t *testing.T) {
	actual := new(bytes.Buffer)
	RootCmd := OsCmds()
	RootCmd.SetOut(actual)
	RootCmd.SetErr(actual)
	RootCmd.SetArgs([]string{"delete", "--addr=localhost:50056", "--insecure", "--resource-id=123"})
	err := RootCmd.Execute()
	assert.Error(t, err)
}

func TestOsCmds_Update(t *testing.T) {
	actual := new(bytes.Buffer)
	RootCmd := OsCmds()
	RootCmd.SetOut(actual)
	RootCmd.SetErr(actual)
	RootCmd.SetArgs([]string{"update", "--addr=localhost:50057", "--insecure", "--resource-id=123"})
	err := RootCmd.Execute()
	assert.Error(t, err)
}
