package controller

import (
	"fmt"
	"os"
	"testing"

	ccev1 "github.com/cnrancher/cce-operator/pkg/apis/cce.pandaria.io/v1"
	"github.com/cnrancher/cce-operator/pkg/huawei/cce"
	"github.com/cnrancher/cce-operator/pkg/huawei/common"
	"github.com/cnrancher/cce-operator/pkg/utils"
	huawei_cce "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/cce/v3"
	"github.com/stretchr/testify/assert"
)

var (
	client *huawei_cce.CceClient
)

func init() {
	accessKey := os.Getenv("HUAWEI_ACCESS_KEY")
	secretKey := os.Getenv("HUAWEI_SECRET_KEY")
	projectID := os.Getenv("HUAWEI_PROJECT_ID")
	if accessKey == "" || secretKey == "" || projectID == "" {
		fmt.Println("skip test CCE")
		return
	}
	auth := common.NewClientAuth(accessKey, secretKey, "cn-north-1", projectID)
	client = cce.NewCCEClient(auth)
}

func Test_CompareNode(t *testing.T) {
	a := ccev1.CCENodePool{}
	b := ccev1.CCENodePool{}
	assert := assert.New(t)
	assert.True(CompareNodePool(&a, &b))
	a = ccev1.CCENodePool{
		Name: "rancher-managed-node-abcde",
		ID:   "abcde-12345",
	}
	a.NodeTemplate = ccev1.CCENodeTemplate{
		Flavor:        "t6.large.2",
		AvailableZone: "cn-north-1a",
		SSHKey:        "test-ssh-key",
		RootVolume: ccev1.CCENodeVolume{
			Size: 40,
			Type: "SSD",
		},
		DataVolumes: []ccev1.CCENodeVolume{
			{
				Size: 100,
				Type: "SSD",
			},
		},
		BillingMode:     0,
		OperatingSystem: "EulerOS 2.9",
		PublicIP: ccev1.CCENodePublicIP{
			Count: 0,
		},
		ExtendParam: ccev1.CCENodeExtendParam{},
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
	b.NodeTemplate.RootVolume = ccev1.CCENodeVolume{
		Size: 50,
		Type: "SSD",
	}
	assert.False(CompareNodePool(&a, &b))
	b = a
	b.NodeTemplate.DataVolumes = nil
	assert.False(CompareNodePool(&a, &b))
	b = a
	b.NodeTemplate.DataVolumes = []ccev1.CCENodeVolume{
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

func Test_BuildUpstreamClusterState(t *testing.T) {
	if client == nil {
		return
	}
	cluster, err := cce.GetCluster(client, "")
	if err != nil {
		t.Error(err)
		return
	}
	nodepools, err := cce.GetClusterNodePools(client, "", false)
	if err != nil {
		t.Error(err)
		return
	}
	state, err := BuildUpstreamClusterState(cluster, nodepools)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("%v\n", utils.PrintObject(state))
}

func Test_BuildUpstreamNodePoolConfigs(t *testing.T) {
	if client == nil {
		return
	}
	res, err := cce.GetClusterNodePools(client, "", false)
	if err != nil {
		t.Error(err)
		return
	}
	pools, err := BuildUpstreamNodePoolConfigs(res)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("%v\n", utils.PrintObject(pools))
}
