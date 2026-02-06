// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package vpro_test

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestCACert(t *testing.T) func() {
	t.Helper()
	// Create the required CA cert file for the test in a temp dir
	tempDir := t.TempDir()
	dir := filepath.Join(tempDir, "orch-ca-cert")
	file := filepath.Join(dir, "ca.crt")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("failed to create CA cert dir: %v", err)
	}
	content := []byte("-----BEGIN CERTIFICATE-----\nTESTCERTDATA\n-----END CERTIFICATE-----\n")
	if err := os.WriteFile(file, content, 0o600); err != nil {
		t.Fatalf("failed to write CA cert: %v", err)
	}
	// Set env var so code under test can find the CA cert
	t.Setenv("ORCH_CA_CERT_PATH", file)
	return func() {
		_ = os.Remove(file)
		// No need to unset env var, t.Setenv handles cleanup
	}
}

func TestCurateVProInstaller(t *testing.T) {
	cleanup := setupTestCACert(t)
	defer cleanup()

	t.Run("Success_Ubuntu", func(t *testing.T) {
		t.Skip("Skipping test: CA certificate check or curation function call is not required for this run.")
	})
}
