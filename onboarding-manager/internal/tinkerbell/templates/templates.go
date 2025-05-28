package templates

import _ "embed"

//go:embed microvisor.yaml
var MicrovisorTemplate []byte
var MicrovisorName = "microvisor"

var UbuntuTemplate []byte
var UbuntuTemplateName = "ubuntu"

var TemplatesMap = map[string][]byte{
	MicrovisorName:     MicrovisorTemplate,
	UbuntuTemplateName: UbuntuTemplate,
}

var (
	OSProfileToTemplateName = map[string]string{
		"microvisor-nonrt":             MicrovisorName,
		"microvisor-rt":                MicrovisorName,
		"ubuntu-24.04-lts-generic":     UbuntuTemplateName,
		"ubuntu-22.04-lts-generic":     UbuntuTemplateName,
		"ubuntu-22.04-lts-generic-ext": UbuntuTemplateName,
	}
)
