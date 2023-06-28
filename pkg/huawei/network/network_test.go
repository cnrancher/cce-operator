package network_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/cnrancher/cce-operator/pkg/huawei/common"
	"github.com/cnrancher/cce-operator/pkg/huawei/network"
	vpc "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/vpc/v2"
	vpcep "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/vpcep/v1"
	"github.com/stretchr/testify/assert"
)

var (
	client       *vpc.VpcClient
	vpcep_client *vpcep.VpcepClient
)

func init() {
	accessKey := os.Getenv("HUAWEI_ACCESS_KEY")
	secretKey := os.Getenv("HUAWEI_SECRET_KEY")
	projectID := os.Getenv("HUAWEI_PROJECT_ID")
	if accessKey == "" || secretKey == "" || projectID == "" {
		fmt.Println("skip test network")
		return
	}
	auth := common.NewClientAuth(accessKey, secretKey, "cn-north-1", projectID)
	client = network.NewVpcClient(auth)
	vpcep_client = network.NewVpcepClient(auth)
}

func Test_GetVpcRoutes(t *testing.T) {
	if client == nil {
		fmt.Println("skip Test_GetVpcRoutes")
		return
	}
	// Modift VPC ID here
	routes, err := network.GetVpcRoutes(
		client, "")
	if err != nil {
		t.Error(err)
		return
	}
	assert.NotNil(t, routes.Routes)
	if t.Failed() {
		return
	}
	fmt.Printf("vpc have %d routes\n", len(*routes.Routes))
	for _, route := range *routes.Routes {
		fmt.Printf("route: %s", route.String())
	}
}

func Test_GetVpcRoute(t *testing.T) {
	if client == nil {
		fmt.Println("skip Test_GetVpcRoute")
		return
	}
	// Modify RouterID here
	route, err := network.GetVpcRoute(client, "")
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("route: %v", route.String())
}

func Test_GetVpc(t *testing.T) {
	if client == nil {
		fmt.Println("skip Test_GetVpc")
		return
	}
	// Modift VPC ID here
	vpc, err := network.GetVPC(client, "216f22a8-59d9-47aa-b5bb-4b5a95ded162")
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("vpc: %v", vpc.String())
}

func Test_GetRouteTables(t *testing.T) {
	if client == nil {
		fmt.Printf("skip Test_GetRouteTables\n")
		return
	}
	rts, err := network.GetRouteTables(client, "", "", "")
	if err != nil {
		t.Error(err)
		return
	}
	for _, rt := range *rts.Routetables {
		fmt.Printf("routeTable: %v", rt.String())
	}
}

func Test_GetVpcepServices(t *testing.T) {
	if vpcep_client == nil {
		fmt.Println("skip Test_GetVpcepServices")
		return
	}
	vpceps, err := network.GetVpcepServices(vpcep_client, "")
	if err != nil {
		t.Error(err)
		return
	}
	for _, vpcep := range *vpceps.EndpointServices {
		fmt.Printf("vpcep: %v\n", vpcep.String())
	}
}
