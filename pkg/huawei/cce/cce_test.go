package cce_test

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/cnrancher/cce-operator/pkg/huawei/cce"
	"github.com/cnrancher/cce-operator/pkg/huawei/common"
	"github.com/cnrancher/cce-operator/pkg/utils"
	huawei_cce "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/cce/v3"
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

func Test_ListNodes(t *testing.T) {
	if client == nil {
		return
	}
	nodes, err := cce.ListNodes(client, "")
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("nodes.Items: \n%v\n", utils.PrintObject(nodes.Items))
}

func Test_ShowCluster(t *testing.T) {
	if client == nil {
		return
	}
	cluster, err := cce.ShowCluster(client, "")
	if err != nil {
		t.Error(err)
		return
	}
	o, _ := json.MarshalIndent(cluster, "", "    ")
	fmt.Printf("cluster: \n%v\n", string(o))
}

func Test_ListNodePools(t *testing.T) {
	if client == nil {
		return
	}
	nodePools, err := cce.ListNodePools(client, "", true)
	if err != nil {
		t.Error(err)
		return
	}
	o := utils.PrintObject(nodePools)
	fmt.Printf("%v\n", o)
}

func Test_GetClusterCert(t *testing.T) {
	if client == nil {
		return
	}
	certs, err := cce.GetClusterCert(client, "", 0)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("%s\n", utils.PrintObject(certs))
}
