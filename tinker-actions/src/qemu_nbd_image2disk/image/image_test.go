// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package image

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWrites(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	t.Run("HTTP request creation fails", func(t *testing.T) {
		ctx := context.Background()
		invalidURL := string([]byte{0x7f}) // Invalid URL
		err := Write(ctx, logger, invalidURL, "/dev/null", false, time.Second)
		if err == nil {
			t.Fatal("Expected error for invalid URL")
		}
	})

	t.Run("HTTP request fails", func(t *testing.T) {
		ctx := context.Background()
		err := Write(ctx, logger, "http://127.0.0.1:1", "/dev/null", false, time.Second)
		if err == nil {
			t.Fatal("Expected error for failed HTTP request")
		}
	})

	t.Run("HTTP status not 200", func(t *testing.T) {
		server := httptest.NewServer(http.NotFoundHandler())
		defer server.Close()

		ctx := context.Background()
		err := Write(ctx, logger, server.URL, "/dev/null", false, time.Second)
		if err == nil || !strings.Contains(err.Error(), "HTTP status code: 404") {
			t.Fatalf("Expected 404 status error, got: %v", err)
		}
	})

	t.Run("SHA mismatch", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("test image content"))
		}))
		defer server.Close()

		// Set invalid expected SHA256
		os.Setenv("SHA256", "invalidsha")
		defer os.Unsetenv("SHA256")

		ctx := context.Background()
		err := Write(ctx, logger, server.URL, "/dev/null", false, time.Second)
		if err == nil || !strings.Contains(err.Error(), "Image SHA-256 hash mismatch") {
			t.Fatalf("Expected SHA mismatch, got: %v", err)
		}
	})

	t.Run("qemu-nbd fails", func(t *testing.T) {
		// Fake qemu-nbd by shadowing it in PATH
		tmpDir := t.TempDir()
		fakeQemu := filepath.Join(tmpDir, "qemu-nbd")
		os.WriteFile(fakeQemu, []byte("#!/bin/sh\nexit 1"), 0755)

		// Prepend to PATH
		oldPath := os.Getenv("PATH")
		os.Setenv("PATH", tmpDir+":"+oldPath)
		defer os.Setenv("PATH", oldPath)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("test image content"))
		}))
		defer server.Close()

		ctx := context.Background()
		err := Write(ctx, logger, server.URL, "/dev/null", false, time.Second)
		if err == nil || !strings.Contains(err.Error(), "network block device attach failed") {
			t.Fatalf("Expected qemu-nbd failure, got: %v", err)
		}
	})

	t.Run("dd fails", func(t *testing.T) {
		// Fake qemu-nbd success
		tmpDir := t.TempDir()
		fakeQemu := filepath.Join(tmpDir, "qemu-nbd")
		os.WriteFile(fakeQemu, []byte("#!/bin/sh\nexit 0"), 0755)

		// Fake dd failure
		fakeDD := filepath.Join(tmpDir, "dd")
		os.WriteFile(fakeDD, []byte("#!/bin/sh\nexit 1"), 0755)

		oldPath := os.Getenv("PATH")
		os.Setenv("PATH", tmpDir+":"+oldPath)
		defer os.Setenv("PATH", oldPath)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("test image content"))
		}))
		defer server.Close()

		ctx := context.Background()
		err := Write(ctx, logger, server.URL, "/dev/null", false, time.Second)
		if err == nil || !strings.Contains(err.Error(), "failed to write image to disk") {
			t.Fatalf("Expected dd failure, got: %v", err)
		}
	})

	t.Run("happy path with no SHA mismatch", func(t *testing.T) {
		// Fake qemu-nbd and dd success
		tmpDir := t.TempDir()
		writeFake := func(name string) {
			_ = os.WriteFile(filepath.Join(tmpDir, name), []byte("#!/bin/sh\nexit 0"), 0755)
		}
		writeFake("qemu-nbd")
		writeFake("dd")

		oldPath := os.Getenv("PATH")
		os.Setenv("PATH", tmpDir+":"+oldPath)
		defer os.Setenv("PATH", oldPath)

		content := []byte("test image")
		sum := sha256.Sum256(content)
		os.Setenv("SHA256", hex.EncodeToString(sum[:]))
		defer os.Unsetenv("SHA256")

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write(content)
		}))
		defer server.Close()
		Write(context.Background(), logger, server.URL, "/dev/null", false, time.Second)

	})
}
