// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: LicenseRef-Intel

package fdoclient

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/common"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/env"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/onboardingmgr/utils"
	inv_errors "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/errors"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/logging"
)

var DefaultClient = NewFDOClient()

const (
	apiUser = "apiUser"

	ContentTypeTextPlain       = "text/plain"
	CertificateAttestationType = "SECP256R1"
)

// HTTPClient used to hide external library under interface to enable testing.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

var (
	clientName = "FDOClient"
	zlog       = logging.GetLogger(clientName)

	httpClient HTTPClient = http.DefaultClient
)

type Client struct {
	// OwnerSvc is the FDO Owner Service API endpoint.
	OwnerSvc string
	// MfgSvc is the FDO Manufacturer Service API endpoint.
	MfgSvc string
}

func NewFDOClient() *Client {
	return &Client{
		OwnerSvc: fmt.Sprintf("http://%s:%s/api/v1", env.FdoOwnerDNS, env.FdoOwnerPort),
		MfgSvc:   fmt.Sprintf("http://%s:%s/api/v1", env.FdoMfgDNS, env.FdoMfgPort),
	}
}

func doAPICall(ctx context.Context, apiURL, httpMethod, apiUser, contentType string, body []byte) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		reader := bytes.NewReader(body)
		reqBody = io.NopCloser(reader)
	}

	req, err := http.NewRequestWithContext(ctx, httpMethod, apiURL, reqBody)
	if err != nil {
		zlog.MiErr(err).Msg("")
		return nil, inv_errors.Errorf("Failed to perform %s API call to %s", httpMethod, apiURL)
	}
	if contentType != "" {
		req.Header.Add("Content-Type", contentType)
	}

	req.SetBasicAuth(apiUser, "")

	resp, err := httpClient.Do(req)
	if err != nil {
		zlog.MiErr(err).Msg("")
		return nil, inv_errors.Errorf("Failed to perform %s API call to %s", httpMethod, apiURL)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		err = inv_errors.Errorf("Failed to perform %s API call to %s with status code %v",
			httpMethod, apiURL, resp.StatusCode)
		zlog.MiErr(err).Msg("")
		return nil, err
	}

	return resp, nil
}

func (f *Client) sendFileToOwner(ctx context.Context, filename, content string) error {
	url := fmt.Sprintf("%s/owner/resource?filename=%s", f.OwnerSvc, filename)

	zlog.Debug().Msgf("Sending file %s to FDO owner", filename)

	resp, err := doAPICall(ctx, url, http.MethodPost, apiUser, ContentTypeTextPlain, []byte(content))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	zlog.Debug().Msgf("File %s successfully sent to FDO owner via resource API", filename)
	return nil
}

func (f *Client) executeSVI(ctx context.Context, content string) error {
	url := fmt.Sprintf("%s/owner/svi", f.OwnerSvc)

	zlog.MiSec().Debug().Msgf("Executing SVI with payload %s", content)

	resp, err := doAPICall(ctx, url, http.MethodPost, apiUser, ContentTypeTextPlain, []byte(content))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	zlog.Debug().Msg("Owner SVI API executed successfully")

	return nil
}

func (f *Client) getOwnerCertificate(ctx context.Context) ([]byte, error) {
	url := fmt.Sprintf("%s/certificate?alias=%s", f.OwnerSvc, CertificateAttestationType)

	zlog.Debug().Msgf("Getting owner certificate from URL %s", url)

	resp, err := doAPICall(ctx, url, http.MethodGet, apiUser, "", []byte{})
	if err != nil {
		return nil, err
	}

	if resp.Body == nil {
		// this should not happen, but just in case
		return nil, inv_errors.Errorf("Failed to read owner certificate: received empty body")
	}
	defer resp.Body.Close()

	ownerCertificate, err := io.ReadAll(resp.Body)
	if err != nil {
		zlog.MiSec().MiErr(err).Msg("")
		return nil, inv_errors.Errorf("Failed to read owner certificate")
	}

	zlog.Debug().Msgf("Got owner certificate")
	return ownerCertificate, nil
}

func (f *Client) getVoucherExtensionFromMfg(ctx context.Context, ownerCertificate []byte, serialNum string) ([]byte, error) {
	url := fmt.Sprintf("%s/mfg/vouchers/%s", f.MfgSvc, serialNum)

	zlog.Debug().Msgf("Getting voucher extension for %s from URL %s", serialNum, url)

	resp, err := doAPICall(ctx, url, http.MethodPost, apiUser, ContentTypeTextPlain, ownerCertificate)
	if err != nil {
		return nil, err
	}

	if resp.Body == nil {
		// this should not happen, but just in case
		return nil, inv_errors.Errorf("Failed to get voucher extension: received empty body")
	}
	defer resp.Body.Close()

	voucherExt, err := io.ReadAll(resp.Body)
	if err != nil {
		zlog.MiSec().MiErr(err).Msg("")
		return nil, inv_errors.Errorf("Failed to read voucher extension")
	}

	zlog.Debug().Msgf("Got voucher extension for serial number %s", serialNum)
	return voucherExt, nil
}

func (f *Client) uploadVoucherExtensionToOwner(ctx context.Context, voucherExt []byte) (string, error) {
	url := fmt.Sprintf("%s/owner/vouchers", f.MfgSvc)

	zlog.Debug().Msgf("Uploading voucher extension to URL %s", url)

	resp, err := doAPICall(ctx, url, http.MethodPost, apiUser, ContentTypeTextPlain, voucherExt)
	if err != nil {
		return "", err
	}

	if resp.Body == nil {
		// this should not happen, but just in case
		return "", inv_errors.Errorf("Failed to get voucher extension: received empty body")
	}
	defer resp.Body.Close()

	fdoGUID, err := io.ReadAll(resp.Body)
	if err != nil {
		zlog.MiSec().MiErr(err).Msg("")
		return "", inv_errors.Errorf("Failed to upload voucher extension")
	}

	zlog.Debug().Msgf("Uploaded voucher extension to owner, got FDO GUID %v", string(fdoGUID))

	return string(fdoGUID), nil
}

func (f *Client) startTO0Process(ctx context.Context, fdoGUID string) error {
	url := fmt.Sprintf("%s/to0/%s", f.OwnerSvc, fdoGUID)

	zlog.Debug().Msgf("Starting TO0 process for FDO GUID %s", fdoGUID)

	resp, err := doAPICall(ctx, url, http.MethodGet, apiUser, ContentTypeTextPlain, []byte(fdoGUID))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (f *Client) doVoucherExtension(ctx context.Context, deviceInfo utils.DeviceInfo) (string, error) {
	zlog.Debug().Msgf("Performing voucher extension for host %s with serial number %s",
		deviceInfo.GUID, deviceInfo.HwSerialID)

	ownerCerificate, err := f.getOwnerCertificate(ctx)
	if err != nil {
		return "", err
	}

	voucherExt, err := f.getVoucherExtensionFromMfg(ctx, ownerCerificate, deviceInfo.HwSerialID)
	if err != nil {
		return "", err
	}

	fdoGUID, err := f.uploadVoucherExtensionToOwner(ctx, voucherExt)
	if err != nil {
		return "", err
	}

	if *common.FlagRVEnabled {
		if err := f.startTO0Process(ctx, fdoGUID); err != nil {
			return "", err
		}
	}

	return fdoGUID, nil
}

func DoVoucherExtension(ctx context.Context, deviceInfo utils.DeviceInfo) (string, error) {
	return DefaultClient.doVoucherExtension(ctx, deviceInfo)
}

func ExecuteSVI(ctx context.Context, content string) error {
	return DefaultClient.executeSVI(ctx, content)
}

func SendFileToOwner(ctx context.Context, filename, content string) error {
	return DefaultClient.sendFileToOwner(ctx, filename, content)
}
