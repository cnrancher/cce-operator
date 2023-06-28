package v1

import (
	"encoding/json"
	"fmt"
	"testing"
)

func Test_CCEClusterConfig_Create(t *testing.T) {
	c := CCEClusterConfig{
		Spec: CCEClusterConfigSpec{
			CredentialSecret: "cattle-global-data:cc-test",
			Category:         "CCE",
			ClusterID:        "",
			Imported:         false,
			Name:             "cce-create-1",
			Labels: map[string]string{
				"key":  "value",
				"key2": "value2",
			},
			Type:        "VirtualMachine",
			Flavor:      "cce.s1.small",
			Version:     "v1.23",
			Description: "example create",
			Ipv6Enable:  false,
			HostNetwork: HostNetwork{
				VpcID:         "<VPC-ID>",
				SubnetID:      "<SUBNET-ID>",
				SecurityGroup: "<SECURITY-GROUP-ID>",
			},
			ContainerNetwork: ContainerNetwork{
				Mode: "overlay_l2",
				CIDR: "172.16.123.0/24",
				// CIDRs: []string{
				// 	"172.16.123.0/24",
				// },
			},
			EniNetwork: EniNetwork{
				Subnets: []string{},
			},
			Authentication: Authentication{
				Mode: "rbac",
				AuthenticatingProxy: AuthenticatingProxy{
					Ca: "",
				},
			},
			BillingMode:          int32(0),
			KubernetesSvcIPRange: "10.3.4.0/24",
			Tags: map[string]string{
				"cluster-key": "cluster-value",
			},
			KubeProxyMode: "",
			NodeConfigs: []NodeConfig{
				{
					Name:          "",
					NodeID:        "",
					Flavor:        "t6.large.2",
					AvailableZone: "cn-north-1a",
					SSHKey:        "",
					RootVolume: Volume{
						Size: 40,
						Type: "SSD",
					},
					DataVolumes: []Volume{
						{
							Size: 100,
							Type: "SSD",
						},
					},
					BillingMode:     0,
					OperatingSystem: "EulerOS 2.9",
					PublicIP: PublicIP{
						Count: 1,
						Eip: Eip{
							Iptype: "5_bgp",
							Bandwidth: Bandwidth{
								ChargeMode: "traffic",
								Size:       1,
								ShareType:  "PER",
							},
						},
					},
					ExtendParam: ExtendParam{
						BMSPeriodType:  "month",
						BMSPeriodNum:   1,
						BMSIsAutoRenew: "false",
					},
					Count: int32(1),
					Labels: map[string]string{
						"node-key":  "value",
						"node-key2": "value2",
					},
				},
			},
		},
	}

	o, e := json.MarshalIndent(c, "", "  ")
	if e != nil {
		t.Error(e)
		return
	}
	fmt.Print(string(o))
}
