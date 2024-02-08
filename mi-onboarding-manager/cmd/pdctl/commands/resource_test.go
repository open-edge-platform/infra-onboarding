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

func TestInstanceResCmds(t *testing.T) {
	actual := new(bytes.Buffer)
	RootCmd := InstanceResCmds()
	RootCmd.SetOut(actual)
	RootCmd.SetErr(actual)
	RootCmd.SetArgs([]string{"get", "--addr=localhost:50058", "--insecure"})
	err := RootCmd.Execute()
	assert.Error(t, err)
}

func TestInstanceResCmds_GetById(t *testing.T) {
	actual := new(bytes.Buffer)
	RootCmd := InstanceResCmds()
	RootCmd.SetOut(actual)
	RootCmd.SetErr(actual)
	RootCmd.SetArgs([]string{"getById", "--addr=localhost:50059", "--insecure", "resource-id=123"})
	err := RootCmd.Execute()
	assert.Error(t, err)
}

func TestInstanceResCmds_Create(t *testing.T) {
	actual := new(bytes.Buffer)
	RootCmd := InstanceResCmds()
	RootCmd.SetOut(actual)
	RootCmd.SetErr(actual)
	RootCmd.SetArgs([]string{
		"create", "--addr=localhost:50061", "--insecure",
		"resource-id=123", "--hostID=123", "--kind=123", "--osID=123",
	})
	err := RootCmd.Execute()
	assert.Error(t, err)
}

func TestInstanceResCmds_Update(t *testing.T) {
	actual := new(bytes.Buffer)
	RootCmd := InstanceResCmds()
	RootCmd.SetOut(actual)
	RootCmd.SetErr(actual)
	RootCmd.SetArgs([]string{"update", "--addr=localhost:50062", "--insecure", "--fields=123", "--resource-id=123"})
	err := RootCmd.Execute()
	assert.Error(t, err)
}

func TestInstanceResCmds_Delate(t *testing.T) {
	actual := new(bytes.Buffer)
	RootCmd := InstanceResCmds()
	RootCmd.SetOut(actual)
	RootCmd.SetErr(actual)
	RootCmd.SetArgs([]string{"delete", "--addr=localhost:50063", "--insecure", "--resource-id=123"})
	err := RootCmd.Execute()
	assert.Error(t, err)
}

func TestHostResCmds(t *testing.T) {
	actual := new(bytes.Buffer)
	RootCmd := HostResCmds()
	RootCmd.SetOut(actual)
	RootCmd.SetErr(actual)
	RootCmd.SetArgs([]string{"get", "--addr=localhost:50064", "--insecure"})
	err := RootCmd.Execute()
	assert.Error(t, err)
}

func TestHostResCmds_getById(t *testing.T) {
	actual := new(bytes.Buffer)
	RootCmd := HostResCmds()
	RootCmd.SetOut(actual)
	RootCmd.SetErr(actual)
	RootCmd.SetArgs([]string{"getById", "--addr=localhost:50065", "--insecure", "--resource-id=123"})
	err := RootCmd.Execute()
	assert.Error(t, err)
}

func TestHostResCmds_getByUUID(t *testing.T) {
	actual := new(bytes.Buffer)
	RootCmd := HostResCmds()
	RootCmd.SetOut(actual)
	RootCmd.SetErr(actual)
	RootCmd.SetArgs([]string{"getByUUID", "--addr=localhost:50066", "--insecure", "--uuid=123"})
	err := RootCmd.Execute()
	assert.Error(t, err)
}

func TestHostResCmds_Create(t *testing.T) {
	actual := new(bytes.Buffer)
	RootCmd := HostResCmds()
	RootCmd.SetOut(actual)
	RootCmd.SetErr(actual)
	RootCmd.SetArgs([]string{
		"create", "--addr=localhost:50067", "--insecure",
		"--hostname=hostname", "--sut-ip=123", "--uuid=123",
	})
	err := RootCmd.Execute()
	assert.Error(t, err)
}

func TestHostResCmds_Update(t *testing.T) {
	actual := new(bytes.Buffer)
	RootCmd := HostResCmds()
	RootCmd.SetOut(actual)
	RootCmd.SetErr(actual)
	RootCmd.SetArgs([]string{"update", "--addr=localhost:50068", "--insecure", "--resource-id=123"})
	err := RootCmd.Execute()
	assert.Error(t, err)
}

func TestHostResCmds_Delete(t *testing.T) {
	actual := new(bytes.Buffer)
	RootCmd := HostResCmds()
	RootCmd.SetOut(actual)
	RootCmd.SetErr(actual)
	RootCmd.SetArgs([]string{"delete", "--addr=localhost:50069", "--insecure", "--resource-id=123"})
	err := RootCmd.Execute()
	assert.Error(t, err)
}
