package persistence

import (
	"testing"

	pb "github.com/intel-sandbox/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/api/grpc/onboardingmgr"
	"github.com/stretchr/testify/assert"
)

func TestMarshalToStr(t *testing.T) {
	hwDetail := []*pb.HwData{{
		HwId:  "xxx",
		MacId: "yyy",
	}}
	hwDetailStr, err := MarshalToStr(hwDetail)
	assert.Equal(t, nil, err)
	assert.Equal(t, "[{\"hw_id\":\"xxx\",\"mac_id\":\"yyy\"}]\n", hwDetailStr)

	onbParams := &pb.OnboardingParams{PdIp: "xxx", PdMac: "yyy"}
	onbParamsStr, err := MarshalToStr(onbParams)
	assert.Equal(t, nil, err)
	assert.Equal(t, "{\"pd_ip\":\"xxx\",\"pd_mac\":\"yyy\"}\n", onbParamsStr)

	obp, err := UnmarshalOnboardingParams(onbParamsStr)
	assert.Equal(t, nil, err)
	assert.Equal(t, "xxx", obp.PdIp)
}
