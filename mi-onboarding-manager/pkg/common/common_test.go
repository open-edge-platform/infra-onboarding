package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type example struct {
	Id  int    `json:"id"`
	Tag string `json:"tag"`
}

func TestGetFields(t *testing.T) {
	fields := GetFields(example{})
	assert.Equal(t, []string{"id", "tag"}, fields)
}
