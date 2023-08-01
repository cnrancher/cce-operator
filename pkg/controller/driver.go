package controller

import (
	"fmt"

	ccev1 "github.com/cnrancher/cce-operator/pkg/apis/cce.pandaria.io/v1"
	"github.com/cnrancher/cce-operator/pkg/huawei/cce"
	"github.com/cnrancher/cce-operator/pkg/huawei/common"
	"github.com/cnrancher/cce-operator/pkg/huawei/dns"
	"github.com/cnrancher/cce-operator/pkg/huawei/eip"
	"github.com/cnrancher/cce-operator/pkg/huawei/elb"
	"github.com/cnrancher/cce-operator/pkg/huawei/nat"
	"github.com/cnrancher/cce-operator/pkg/huawei/vpc"
	"github.com/cnrancher/cce-operator/pkg/huawei/vpcep"
	"github.com/cnrancher/cce-operator/pkg/utils"
	huawei_cce "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/cce/v3"
	huawei_dns "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/dns/v2"
	huawei_eip "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/eip/v2"
	huawei_elb "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/elb/v2"
	huawei_nat "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/nat/v2"
	huawei_vpc "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/vpc/v2"
	huawei_vpcep "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/vpcep/v1"
	wranglerv1 "github.com/rancher/wrangler/pkg/generated/controllers/core/v1"
)

type HuaweiDriver struct {
	VPC   *huawei_vpc.VpcClient
	EIP   *huawei_eip.EipClient
	ELB   *huawei_elb.ElbClient
	CCE   *huawei_cce.CceClient
	VPCEP *huawei_vpcep.VpcepClient
	DNS   *huawei_dns.DnsClient
	NAT   *huawei_nat.NatClient
}

func NewHuaweiDriver(auth *common.ClientAuth) *HuaweiDriver {
	return &HuaweiDriver{
		CCE:   cce.NewCCEClient(auth),
		ELB:   elb.NewElbClient(auth),
		VPC:   vpc.NewVpcClient(auth),
		EIP:   eip.NewEipClient(auth),
		VPCEP: vpcep.NewVpcepClient(auth),
		DNS:   dns.NewDnsClient(auth),
		NAT:   nat.NewNatClient(auth),
	}
}

func (h *Handler) newDriver(secretsCache wranglerv1.SecretCache, spec *ccev1.CCEClusterConfigSpec) error {
	auth, err := NewHuaweiClientAuth(secretsCache, spec)
	if err != nil {
		return err
	}
	h.driver = *NewHuaweiDriver(auth)

	return nil
}

func NewHuaweiClientAuth(
	secretsCache wranglerv1.SecretCache, spec *ccev1.CCEClusterConfigSpec,
) (*common.ClientAuth, error) {
	region := spec.RegionID
	if region == "" {
		return nil, fmt.Errorf("regionID not provided")
	}
	ns, id := utils.Parse(spec.HuaweiCredentialSecret)
	if spec.HuaweiCredentialSecret == "" {
		return nil, fmt.Errorf("huawei credential secret not provided")
	}

	secret, err := secretsCache.Get(ns, id)
	if err != nil {
		return nil, fmt.Errorf("error getting secret %s/%s: %w", ns, id, err)
	}

	accessKeyBytes := secret.Data["huaweicredentialConfig-accessKey"]
	secretKeyBytes := secret.Data["huaweicredentialConfig-secretKey"]
	projectIDBytes := secret.Data["huaweicredentialConfig-projectID"]
	if accessKeyBytes == nil || secretKeyBytes == nil || projectIDBytes == nil {
		return nil, fmt.Errorf("invalid huawei cloud credential")
	}
	accessKey := string(accessKeyBytes)
	secretKey := string(secretKeyBytes)
	projectID := string(projectIDBytes)
	return common.NewClientAuth(accessKey, secretKey, region, projectID), nil
}
