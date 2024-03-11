/*
   Copyright (C) 2023 Intel Corporation
   SPDX-License-Identifier: Apache-2.0
*/

package persistence

type (
	ArtifactData struct {
		ID          string           `json:"id" db:"id"`
		Category    ArtifactCategory `json:"category" db:"category"`
		Name        string           `json:"name" db:"name"`
		Version     string           `json:"version" db:"version" `
		Description string           `json:"descrip" db:"descrip"`
		Detail      string           `json:"detail" db:"detail"`
		PackageURL  string           `json:"pkg_url" db:"pkg_url"`
		Author      string           `json:"author" db:"author"`
		State       string           `json:"state" db:"state"`
		License     string           `json:"license" db:"license"`
	}

	NodeData struct {
		ID               string `json:"id" db:"id"`
		HwID             string `json:"hwid" db:"hwid"`
		PlatformType     string `json:"plat_type" db:"plat_type"`
		FwArtID          string `json:"fw_art_id" db:"fw_art_id"`
		OsArtID          string `json:"os_art_id" db:"os_art_id"`
		AppArtID         string `json:"app_art_id" db:"app_art_id"`
		PlatformArtID    string `json:"plat_art_id" db:"plat_art_id"`
		DeviceType       string `json:"dev_type" db:"dev_type"`
		DeviceInfoAgent  string `json:"dev_info_agent" db:"dev_info_agent"`
		DeviceStatus     string `json:"dev_status" db:"dev_status"`
		UpdateStatus     string `json:"update_status" db:"update_status"`
		UpdateAvailable  string `json:"update_avl" db:"update_avl"`
		OnboardingStatus string `json:"onboard_status" db:"onboard_status"`
	}

	ArtifactCategory string
)

const (
	Platform    ArtifactCategory = "platform"
	Bios        ArtifactCategory = "bios"
	Os          ArtifactCategory = "os"
	Application ArtifactCategory = "app"
	Container   ArtifactCategory = "container"
)

func (ac ArtifactCategory) String() string {
	return string(ac)
}
