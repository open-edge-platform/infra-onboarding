// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: LicenseRef-Intel

package fdoclient

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	grpc_status "google.golang.org/grpc/status"
)

func TestNewFDOClient(t *testing.T) {
	t.Run("Use defaults", func(t *testing.T) {
		fdoClient := NewFDOClient()
		require.NotNil(t, fdoClient)
		assert.Equal(t, fmt.Sprintf("http://%s:%s/api/v1", DefaultOwnerURL, DefaultOwnerPort), fdoClient.OwnerSvc)
		assert.Equal(t, fmt.Sprintf("http://%s:%s/api/v1", DefaultMfgURL, DefaultMfgPort), fdoClient.MfgSvc)
	})

	t.Run("Use env vars", func(t *testing.T) {
		os.Setenv(EnvOwnerURL, "example")
		os.Setenv(EnvOwnerPort, "1234")
		os.Setenv(EnvMfgURL, "example")
		os.Setenv(EnvMfgPort, "5678")

		defer func() {
			os.Unsetenv(EnvOwnerURL)
			os.Unsetenv(EnvOwnerPort)
			os.Unsetenv(EnvMfgURL)
			os.Unsetenv(EnvMfgPort)
		}()

		fdoClient := NewFDOClient()
		require.NotNil(t, fdoClient)
		assert.Equal(t, "http://example:1234/api/v1", fdoClient.OwnerSvc)
		assert.Equal(t, "http://example:5678/api/v1", fdoClient.MfgSvc)
	})
}

func TestDoAPICall(t *testing.T) {
	t.Run("Invalid HTTP method", func(t *testing.T) {
		resp, err := doAPICall(context.Background(), "url", "INVALID METHOD", "", "", nil)
		require.Error(t, err)
		require.Nil(t, resp)
		errInfo := grpc_status.Convert(err)
		assert.Equal(t, codes.Internal, errInfo.Code())
	})

	t.Run("Failed HTTP request", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
		}))
		defer srv.Close()
		resp, err := doAPICall(context.Background(), srv.URL, http.MethodPost, "", ContentTypeTextPlain, []byte("some string"))
		require.Error(t, err)
		require.Nil(t, resp)
		errInfo := grpc_status.Convert(err)
		assert.Equal(t, codes.Internal, errInfo.Code())
	})

	t.Run("Unexpected status code", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()
		resp, err := doAPICall(context.Background(), srv.URL, http.MethodPost, "", ContentTypeTextPlain, []byte("some string"))
		require.Error(t, err)
		require.Nil(t, resp)
		errInfo := grpc_status.Convert(err)
		assert.Equal(t, codes.Internal, errInfo.Code())
	})

	t.Run("Success", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()
		resp, err := doAPICall(context.Background(), srv.URL, http.MethodPost, "", ContentTypeTextPlain, []byte("some string"))
		require.NoError(t, err)
		require.NotNil(t, resp)
	})
}

func TestSendFileToOwner(t *testing.T) {
	t.Run("Failed", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()

		DefaultClient.OwnerSvc = srv.URL
		err := SendFileToOwner(context.Background(), "filename", "some content")
		require.Error(t, err)
	})

	t.Run("Success", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("some string"))
		}))
		defer srv.Close()

		DefaultClient.OwnerSvc = srv.URL
		err := SendFileToOwner(context.Background(), "filename", "some content")
		require.NoError(t, err)
	})
}

func TestExecuteSVI(t *testing.T) {
	t.Run("Failed", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()

		DefaultClient.OwnerSvc = srv.URL
		err := ExecuteSVI(context.Background(), "some content")
		require.Error(t, err)
	})

	t.Run("Success", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("some string"))
		}))
		defer srv.Close()

		DefaultClient.OwnerSvc = srv.URL
		err := ExecuteSVI(context.Background(), "some content")
		require.NoError(t, err)
	})
}

func TestGetOwnerCertificate(t *testing.T) {
	fdoClient := NewFDOClient()
	t.Run("Failed HTTP call", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()

		fdoClient.OwnerSvc = srv.URL
		_, err := fdoClient.getOwnerCertificate(context.Background())
		require.Error(t, err)
	})

	t.Run("Success", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("some string"))
		}))
		defer srv.Close()

		fdoClient.OwnerSvc = srv.URL
		cert, err := fdoClient.getOwnerCertificate(context.Background())
		require.NoError(t, err)
		assert.Equal(t, []byte("some string"), cert)
	})
}

func TestGetVoucherExtensionFromMfg(t *testing.T) {
	ownerCert := []byte("some string")
	fdoClient := NewFDOClient()
	t.Run("Failed HTTP call", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()

		fdoClient.MfgSvc = srv.URL
		_, err := fdoClient.getVoucherExtensionFromMfg(context.Background(), ownerCert, "1234567")
		require.Error(t, err)
	})

	t.Run("Success", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("voucher ext"))
		}))
		defer srv.Close()

		fdoClient.MfgSvc = srv.URL
		voucherExt, err := fdoClient.getVoucherExtensionFromMfg(context.Background(), ownerCert, "1234567")
		require.NoError(t, err)
		assert.Equal(t, []byte("voucher ext"), voucherExt)
	})
}

func TestUploadVoucherExtensionToOwner(t *testing.T) {
	fdoClient := NewFDOClient()
	t.Run("Failed HTTP call", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()

		fdoClient.MfgSvc = srv.URL
		fdoGUID, err := fdoClient.uploadVoucherExtensionToOwner(context.Background(), []byte("voucher ext"))
		require.Error(t, err)
		require.Empty(t, fdoGUID)
	})

	t.Run("Success", func(t *testing.T) {
		testFdoGUID := uuid.NewString()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(testFdoGUID))
		}))
		defer srv.Close()

		fdoClient.MfgSvc = srv.URL
		fdoGUID, err := fdoClient.uploadVoucherExtensionToOwner(context.Background(), []byte("voucher ext"))
		require.NoError(t, err)
		assert.Equal(t, testFdoGUID, fdoGUID)
	})
}

func TestStartTO0Process(t *testing.T) {
	fdoClient := NewFDOClient()
	t.Run("Failed HTTP call", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()

		fdoClient.OwnerSvc = srv.URL
		err := fdoClient.startTO0Process(context.Background(), uuid.NewString())
		require.Error(t, err)
	})

	t.Run("Success", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		fdoClient.OwnerSvc = srv.URL
		err := fdoClient.startTO0Process(context.Background(), uuid.NewString())
		require.NoError(t, err)
	})
}
