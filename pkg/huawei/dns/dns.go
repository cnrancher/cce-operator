package dns

import (
	"github.com/cnrancher/cce-operator/pkg/huawei/common"
	dns "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/dns/v2"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/dns/v2/model"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/dns/v2/region"
	"github.com/sirupsen/logrus"
)

func NewDNSClient(c *common.ClientAuth) *dns.DnsClient {
	return dns.NewDnsClient(
		dns.DnsClientBuilder().
			WithRegion(region.ValueOf(c.Region)).
			WithCredential(c.Credential).
			Build())
}

func ListNameServers(client *dns.DnsClient, region string) (*model.ListNameServersResponse, error) {
	res, err := client.ListNameServers(&model.ListNameServersRequest{
		Region: &region,
	})
	if err != nil {
		logrus.Debugf("ListNameServers failed: Region %q", region)
	}
	return res, err
}
