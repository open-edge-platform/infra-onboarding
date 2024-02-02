/*
 * SPDX-FileCopyrightText: (C) 2023 Intel Corporation
 * SPDX-License-Identifier: LicenseRef-Intel
 */
package commands

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInstanceResourceCmd_Get(t *testing.T) {

	actual := new(bytes.Buffer)
	RootCmd := InstanceResourceCmd()
	RootCmd.SetOut(actual)
	RootCmd.SetErr(actual)
	RootCmd.SetArgs([]string{"get", "--addr=localhost:50051", "--insecure"})

	err := RootCmd.Execute()
	assert.Error(t, err)
}

func TestInstanceResourceCmd_Create(t *testing.T) {

	actual := new(bytes.Buffer)
	RootCmd := InstanceResourceCmd()
	RootCmd.SetOut(actual)
	RootCmd.SetErr(actual)
	RootCmd.SetArgs([]string{"create", "--addr=localhost:50051", "--insecure"})

	err := RootCmd.Execute()
	assert.Error(t, err)
}

func TestInstanceResourceCmd_Update(t *testing.T) {

	actual := new(bytes.Buffer)
	RootCmd := InstanceResourceCmd()
	RootCmd.SetOut(actual)
	RootCmd.SetErr(actual)
	RootCmd.SetArgs([]string{"update", "--addr=localhost:50051", "--insecure", "--artifact_id=123"})

	err := RootCmd.Execute()
	assert.Error(t, err)
}

func TestInstanceResourceCmd_Delete(t *testing.T) {

	actual := new(bytes.Buffer)
	RootCmd := InstanceResourceCmd()
	RootCmd.SetOut(actual)
	RootCmd.SetErr(actual)
	RootCmd.SetArgs([]string{"delete", "--addr=localhost:50051", "--insecure"})

	err := RootCmd.Execute()
	assert.Error(t, err)
}

