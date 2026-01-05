// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package interactive

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"device-discovery/internal/config"
)

// ExecuteAuthScript executes the client-auth.sh script for TTY-based authentication.
// The script prompts the user for Keycloak credentials via TTY devices.
func ExecuteAuthScript(ctx context.Context, scriptContent []byte) error {
	tmpfile, err := config.CreateTempScript(scriptContent)
	if err != nil {
		return fmt.Errorf("error creating temporary file: %w", err)
	}
	defer func() {
		tmpfile.Close()
		os.Remove(tmpfile.Name())
	}()

	cmd := exec.CommandContext(ctx, "/bin/sh", tmpfile.Name())
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error starting command: %w", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				fmt.Printf("STDERR:\n%s\n", string(exitErr.Stderr))
			}
			return fmt.Errorf("error executing command: %w", err)
		}
		fmt.Println("client-auth.sh executed successfully")
		return nil
	case <-ctx.Done():
		fmt.Println("client-auth.sh timed out, killing process group...")
		syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) // Kill the process group
		return fmt.Errorf("client-auth.sh timed out: %w", ctx.Err())
	}
}
