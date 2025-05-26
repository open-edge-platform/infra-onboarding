// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

// NOTICE: This file has been modified by Intel Corporation.
// Original file can be found at https://github.com/tinkerbell/actions.

package image

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ulikunitz/xz"
)

func gzipReader(t *testing.T) io.Reader {
	t.Helper()

	var b bytes.Buffer
	gzW := gzip.NewWriter(&b)
	if _, err := gzW.Write([]byte("YourDataHere")); err != nil {
		t.Fatal(err)
	}
	if err := gzW.Close(); err != nil {
		t.Fatal(err)
	}
	rdata := strings.NewReader(b.String())

	return rdata
}

func xzReader(t *testing.T) io.Reader {
	t.Helper()

	var b bytes.Buffer
	xzW, _ := xz.NewWriter(&b)
	if _, err := xzW.Write([]byte("YourDataHere")); err != nil {
		t.Fatal(err)
	}
	if err := xzW.Close(); err != nil {
		t.Fatal(err)
	}
	rdata := strings.NewReader(b.String())

	return rdata
}

func Test_findDecompressor(t *testing.T) {
	tests := []struct {
		name     string
		imageURL string
		reader   func(*testing.T) io.Reader
		wantOut  io.Reader
		wantErr  bool
	}{
		{
			"tar gzip",
			"http://192.168.0.1/a.tar.gz",
			gzipReader,
			nil,
			false,
		},
		{
			"broken gzip",
			"http://192.168.0.1/a.gz",
			xzReader,
			nil,
			true,
		},
		{
			"xz",
			"http://192.168.0.1/a.xz",
			xzReader,
			nil,
			false,
		},
		{
			"broken gzip",
			"http://192.168.0.1/a.xz",
			gzipReader,
			nil,
			true,
		},
		{
			"unknown",
			"http://192.168.0.1/a.abc",
			xzReader,
			nil,
			true,
		},
		{
			"zs",
			"http://192.168.0.1/a.zs",
			xzReader,
			nil,
			false,
		},
		{
			"bz2",
			"http://192.168.0.1/a.bz2",
			xzReader,
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := findDecompressor(tt.imageURL, tt.reader(t))
			if (err != nil) != tt.wantErr {
				t.Errorf("findDecompressor() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

type errorWriter struct{}

func (e *errorWriter) Write(p []byte) (n int, err error) {
	return 0, errors.New("mock write error")
}

func TestProgress_Write(t *testing.T) {
	type fields struct {
		w      io.Writer
		r      io.Reader
		wBytes atomic.Int64
		rBytes atomic.Int64
	}
	type args struct {
		b []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantN   int
		wantErr bool
	}{
		{
			name: "successful write",
			fields: fields{
				w: &bytes.Buffer{},
			},
			args: args{
				b: []byte("hello"),
			},
			wantN:   5,
			wantErr: false,
		},
		{
			name: "write error",
			fields: fields{
				w: &errorWriter{},
			},
			args: args{
				b: []byte("fail"),
			},
			wantN:   0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Progress{
				w:      tt.fields.w,
				r:      tt.fields.r,
				wBytes: tt.fields.wBytes,
				rBytes: tt.fields.rBytes,
			}
			gotN, err := p.Write(tt.args.b)
			if (err != nil) != tt.wantErr {
				t.Errorf("Progress.Write() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotN != tt.wantN {
				t.Errorf("Progress.Write() = %v, want %v", gotN, tt.wantN)
			}
		})
	}
}

func TestWriteCounter_Write(t *testing.T) {
	type fields struct {
		Total uint64
	}
	type args struct {
		p []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    int
		wantErr bool
	}{
		{
			name: "With write counter",
			fields: fields{
				Total: 0,
			},
			args: args{
				p: []byte{},
			},
			want:    0,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wc := &WriteCounter{
				Total: tt.fields.Total,
			}
			got, err := wc.Write(tt.args.p)
			if (err != nil) != tt.wantErr {
				t.Errorf("WriteCounter.Write() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("WriteCounter.Write() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWrite(t *testing.T) {
	type args struct {
		ctx               context.Context
		log               *slog.Logger
		sourceImage       string
		destinationDevice string
		compressed        bool
		progressInterval  time.Duration
	}

	// Start a fake HTTP server
	fakeImage := []byte("fake image content")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(fakeImage)
	}))
	defer server.Close()

	// Create a temporary destination file
	tmpFile, err := os.CreateTemp("", "test-disk")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Set SHA256 env var for match
	hash := sha256.Sum256(fakeImage)
	os.Setenv("SHA256", hex.EncodeToString(hash[:]))

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "successful write with sha match",
			args: args{
				ctx:               context.Background(),
				log:               slog.New(slog.NewTextHandler(io.Discard, nil)),
				sourceImage:       server.URL,
				destinationDevice: tmpFile.Name(),
				compressed:        false,
				progressInterval:  5 * time.Millisecond,
			},
			wantErr: false,
		},
		{
			name: "404 not found",
			args: args{
				ctx:               context.Background(),
				log:               slog.New(slog.NewTextHandler(io.Discard, nil)),
				sourceImage:       server.URL + "/not-found",
				destinationDevice: tmpFile.Name(),
				compressed:        false,
				progressInterval:  5 * time.Millisecond,
			},
			wantErr: false,
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

type errorReader struct{}

func (e *errorReader) Read(_ []byte) (int, error) {
	return 0, errors.New("forced read error")
}

func TestProgress_Read(t *testing.T) {
	type fields struct {
		w      io.Writer
		r      io.Reader
		wBytes atomic.Int64
		rBytes atomic.Int64
	}
	type args struct {
		b []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantN   int
		wantErr bool
	}{
		{
			name: "successful read",
			fields: fields{
				r: strings.NewReader("hello world"),
			},
			args: args{
				b: make([]byte, 5),
			},
			wantN:   5,
			wantErr: false,
		},
		{
			name: "read error",
			fields: fields{
				r: &errorReader{},
			},
			args: args{
				b: make([]byte, 5),
			},
			wantN:   0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Progress{
				w:      tt.fields.w,
				r:      tt.fields.r,
				wBytes: tt.fields.wBytes,
				rBytes: tt.fields.rBytes,
			}
			gotN, err := p.Read(tt.args.b)
			if (err != nil) != tt.wantErr {
				t.Errorf("Progress.Read() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotN != tt.wantN {
				t.Errorf("Progress.Read() = %v, want %v", gotN, tt.wantN)
			}
		})
	}
}
