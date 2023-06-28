package controller

import (
	"testing"

	ccev1 "github.com/cnrancher/cce-operator/pkg/apis/cce.pandaria.io/v1"
	"github.com/stretchr/testify/assert"
)

func Test_CompareNode(t *testing.T) {
	a := ccev1.NodeConfig{}
	b := ccev1.NodeConfig{}
	assert := assert.New(t)
	assert.True(CompareNode(&a, &b))
	a = ccev1.NodeConfig{
		Name:          "rancher-managed-node-abcde",
		NodeID:        "abcde-12345",
		Flavor:        "t6.large.2",
		AvailableZone: "cn-north-1a",
		SSHKey:        "test-ssh-key",
		RootVolume: ccev1.Volume{
			Size: 40,
			Type: "SSD",
		},
		DataVolumes: []ccev1.Volume{
			{
				Size: 100,
				Type: "SSD",
			},
		},
		BillingMode:     0,
		OperatingSystem: "EulerOS 2.9",
		PublicIP: ccev1.PublicIP{
			Count: 0,
		},
		ExtendParam: ccev1.ExtendParam{},
		Labels: map[string]string{
			"label": "aaa",
		},
		Count: 1,
	}
	b = a
	b.Name = ""
	b.NodeID = ""
	assert.True(CompareNode(&a, &b))
	b.Flavor = "c3.large.2"
	assert.False(CompareNode(&a, &b))
	b = a
	b.AvailableZone = "cn-north-1b"
	assert.False(CompareNode(&a, &b))
	b = a
	b.SSHKey = ""
	assert.False(CompareNode(&a, &b))
	b = a
	b.RootVolume = ccev1.Volume{
		Size: 50,
		Type: "SSD",
	}
	assert.False(CompareNode(&a, &b))
	b = a
	b.DataVolumes = nil
	assert.False(CompareNode(&a, &b))
	b = a
	b.DataVolumes = []ccev1.Volume{
		{
			Size: 110,
			Type: "SSD",
		},
	}
	assert.False(CompareNode(&a, &b))
	b = a
	b.BillingMode = 1
	assert.False(CompareNode(&a, &b))
	b = a
	b.OperatingSystem = "CentOS"
	assert.False(CompareNode(&a, &b))
	// b = a
	// b.PublicIP = ccev1.PublicIP{
	// 	Count: 1,
	// }
	// assert.False(CompareNode(&a, &b))
	// b = a
	// b.ExtendParam = ccev1.ExtendParam{
	// 	BMSPeriodNum: 1,
	// }
	// assert.False(CompareNode(&a, &b))
}
