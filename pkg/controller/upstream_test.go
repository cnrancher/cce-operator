package controller

import (
	"testing"

	ccev1 "github.com/cnrancher/cce-operator/pkg/apis/cce.pandaria.io/v1"
	"github.com/stretchr/testify/assert"
)

func Test_CompareNode(t *testing.T) {
	a := ccev1.NodePool{}
	b := ccev1.NodePool{}
	assert := assert.New(t)
	assert.True(CompareNodePool(&a, &b))
	a = ccev1.NodePool{
		Name: "rancher-managed-node-abcde",
		ID:   "abcde-12345",
	}
	a.NodeTemplate = ccev1.NodeTemplate{
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
		Count:       1,
	}
	b = a
	b.Name = ""
	b.ID = ""
	assert.True(CompareNodePool(&a, &b))
	b.NodeTemplate.Flavor = "c3.large.2"
	assert.False(CompareNodePool(&a, &b))
	b = a
	b.NodeTemplate.AvailableZone = "cn-north-1b"
	assert.False(CompareNodePool(&a, &b))
	b = a
	b.NodeTemplate.SSHKey = ""
	assert.False(CompareNodePool(&a, &b))
	b = a
	b.NodeTemplate.RootVolume = ccev1.Volume{
		Size: 50,
		Type: "SSD",
	}
	assert.False(CompareNodePool(&a, &b))
	b = a
	b.NodeTemplate.DataVolumes = nil
	assert.False(CompareNodePool(&a, &b))
	b = a
	b.NodeTemplate.DataVolumes = []ccev1.Volume{
		{
			Size: 110,
			Type: "SSD",
		},
	}
	assert.False(CompareNodePool(&a, &b))
	b = a
	b.NodeTemplate.BillingMode = 1
	assert.False(CompareNodePool(&a, &b))
	b = a
	b.NodeTemplate.OperatingSystem = "CentOS"
	assert.False(CompareNodePool(&a, &b))
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
