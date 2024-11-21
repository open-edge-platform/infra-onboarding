/* #####################################################################################
# INTEL CONFIDENTIAL                                                                #
# Copyright (C) 2024 Intel Corporation                                              #
# This software and the related documents are Intel copyrighted materials,          #
# and your use of them is governed by the express license under which they          #
# were provided to you ("License"). Unless the License provides otherwise,          #
# you may not use, modify, copy, publish, distribute, disclose or transmit          #
# this software or the related documents without Intel's prior written permission.  #
# This software and the related documents are provided as is, with no express       #
# or implied warranties, other than those that are expressly stated in the License. #
#####################################################################################*/

package image

import (
	"bytes"
	"compress/gzip"
	"io"
	"strings"
	"testing"

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
			"unknown",
			"http://192.168.0.1/a.abc",
			xzReader,
			nil,
			true,
		},
		// TODO: Add test cases.
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
