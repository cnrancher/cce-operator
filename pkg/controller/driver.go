package controller

import (
	"github.com/cnrancher/cce-operator/pkg/huawei/cce"
	"github.com/cnrancher/cce-operator/pkg/huawei/common"
	"github.com/cnrancher/cce-operator/pkg/huawei/elb"
	"github.com/cnrancher/cce-operator/pkg/huawei/network"
	huawei_cce "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/cce/v3"
	huawei_dns "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/dns/v2"
	huawei_eip "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/eip/v2"
	huawei_elb "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/elb/v2"
	huawei_vpc "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/vpc/v2"
	huawei_vpcep "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/vpcep/v1"
)

type HuaweiDriver struct {
	VPC   *huawei_vpc.VpcClient
	EIP   *huawei_eip.EipClient
	ELB   *huawei_elb.ElbClient
	CCE   *huawei_cce.CceClient
	VPCEP *huawei_vpcep.VpcepClient
	DNS   *huawei_dns.DnsClient
}

func NewHuaweiDriver(auth *common.ClientAuth) *HuaweiDriver {
	return &HuaweiDriver{
		CCE:   cce.NewCCEClient(auth),
		ELB:   elb.NewElbClient(auth),
		VPC:   network.NewVpcClient(auth),
		EIP:   network.NewEipClient(auth),
		VPCEP: network.NewVpcepClient(auth),
		DNS:   network.NewDnsClient(auth),
	}
}
