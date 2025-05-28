package templates

import _ "embed"

//go:embed microvisor.yaml
var Microvisor []byte
var MicrovisorName = "microvisor"

var TemplatesMap = map[string][]byte{
	MicrovisorName: Microvisor,
}
