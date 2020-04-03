package cron_hpa

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestController(t *testing.T) {
	var cronCfg[] struct {
		Schedule string `json:"schedule"`
		Replicas uint32 `json:"replicas,omitempty"`
		MinReplicas uint32 `json:"minReplicas,omitempty"`
		MaxReplicas uint32 `json:"maxReplicas,omitempty"`
	}

	err := json.Unmarshal([]byte("[{\"schedule\": \"* * * * *\", \"replicas\": 4}]"), &cronCfg)
	assert.Nil(t, err)
}