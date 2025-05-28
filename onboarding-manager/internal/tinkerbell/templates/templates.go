package templates

import (
	_ "embed"
	osv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/os/v1"
)

//go:embed microvisor.yaml
var MicrovisorTemplate []byte
var MicrovisorName = "microvisor"

//go:embed ubuntu.yaml
var UbuntuTemplate []byte
var UbuntuTemplateName = "ubuntu"

var TemplatesMap = map[string][]byte{
	MicrovisorName:     MicrovisorTemplate,
	UbuntuTemplateName: UbuntuTemplate,
}

var (
	// TODO: This uses OS type now but should be based on OS distro or profile name.
	OSTypeToTemplateName = map[osv1.OsType]string{
		osv1.OsType_OS_TYPE_MUTABLE:   UbuntuTemplateName,
		osv1.OsType_OS_TYPE_IMMUTABLE: MicrovisorName,
	}
)
