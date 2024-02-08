/*
 * SPDX-FileCopyrightText: (C) 2023 Intel Corporation
 * SPDX-License-Identifier: LicenseRef-Intel
 */
package commands

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHostResourceCmd(t *testing.T) {
	actual := new(bytes.Buffer)
	RootCmd := HostResourceCmd()
	RootCmd.SetOut(actual)
	RootCmd.SetErr(actual)
	RootCmd.SetArgs([]string{"add", "--addr=localhost:50051", "--insecure"})
	err := RootCmd.Execute()
	assert.Error(t, err)
}

func TestHostResourceCmd_Get(t *testing.T) {
	actual := new(bytes.Buffer)
	RootCmd := HostResourceCmd()
	RootCmd.SetOut(actual)
	RootCmd.SetErr(actual)
	RootCmd.SetArgs([]string{"get", "--addr=localhost:50051", "--insecure"})
	err := RootCmd.Execute()
	assert.Error(t, err)
}

func TestHostResourceCmd_Update(t *testing.T) {
	actual := new(bytes.Buffer)
	RootCmd := HostResourceCmd()
	RootCmd.SetOut(actual)
	RootCmd.SetErr(actual)
	RootCmd.SetArgs([]string{"update", "--addr=localhost:50051", "--insecure", "--hw-id=123"})
	err := RootCmd.Execute()
	assert.Error(t, err)
}

func TestHostResourceCmd_Delete(t *testing.T) {
	actual := new(bytes.Buffer)
	RootCmd := HostResourceCmd()
	RootCmd.SetOut(actual)
	RootCmd.SetErr(actual)
	RootCmd.SetArgs([]string{"delete", "--addr=localhost:50051", "--insecure"})
	err := RootCmd.Execute()
	assert.Error(t, err)
}

func TestHostResourceCmd_Add(t *testing.T) {
	caCertPath := "/home/" + os.Getenv("USER") + "/.fdo-secrets/scripts/secrets/ca-cert.pem"
	certPath := "/home/" + os.Getenv("USER") + "/.fdo-secrets/scripts/secrets/api-user.pem"
	actual := new(bytes.Buffer)
	RootCmd := HostResourceCmd()
	RootCmd.SetOut(actual)
	RootCmd.SetErr(actual)
	cert := "--cert=" + certPath
	cacert := "--cacert=" + caCertPath
	RootCmd.SetArgs([]string{"add", "--addr=localhost:50051", cert, "--key=123", cacert})
	err := RootCmd.Execute()
	assert.Error(t, err)
}
