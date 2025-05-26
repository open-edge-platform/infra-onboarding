// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package image

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestWrite(t *testing.T) {
	type args struct {
		ctx               context.Context
		log               *slog.Logger
		sourceImage       string
		destinationDevice string
		compressed        bool
		progressInterval  time.Duration
	}

	imageData := []byte("fake image data")
	hash := sha256.Sum256(imageData)
	expectedSHA256 := hex.EncodeToString(hash[:])
	os.Setenv("SHA256", expectedSHA256)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(imageData)
	}))
	defer server.Close()

	tmpDev, err := os.CreateTemp("", "fake-dev-*")
	if err != nil {
		t.Fatalf("failed to create fake device: %v", err)
	}
	tmpDevPath := tmpDev.Name()
	tmpDev.Close()
	defer os.Remove(tmpDevPath)

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "happy path till nbd (mocked)",
			args: args{
				ctx:               context.Background(),
				log:               slog.New(slog.NewTextHandler(io.Discard, nil)),
				sourceImage:       server.URL,
				destinationDevice: tmpDevPath, 
				compressed:        false,
				progressInterval:  time.Second,
			},
			wantErr: true, 
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Write(tt.args.ctx, tt.args.log, tt.args.sourceImage, tt.args.destinationDevice, tt.args.compressed, tt.args.progressInterval)
			if (err != nil) != tt.wantErr {
				t.Errorf("Write() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
