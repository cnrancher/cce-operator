package v1

import (
	"encoding/json"
	"fmt"
	"testing"
)

func Test_CCEClusterConfig_Create(t *testing.T) {
	c := CCEClusterConfig{
		Spec: CCEClusterConfigSpec{
			HuaweiCredentialSecret: "cattle-global-data:cc-xxxx",
			Category:               "CCE",
			ClusterID:              "aaa-bbb-ccc",
			Imported:               false,
			Name:                   "cce-test",
			Labels: map[string]string{
				"key":  "value",
				"key2": "value2",
			},
			Type:        "VirtualMachine",
			Flavor:      "cce.s1.small",
			Version:     "v1.23",
			Description: "example description",
			Ipv6Enable:  false,
			HostNetwork: CCEHostNetwork{
				VpcID:         "VPCID-xxxxxx",
				SubnetID:      "SUBNETID-xxxxxx",
				SecurityGroup: "SECURITY-GROUP-ID-xxxxx",
			},
			ContainerNetwork: CCEContainerNetwork{
				Mode: "overlay_l2",
				CIDR: "172.16.123.0/24",
				// CIDRs: []string{
				// 	"172.16.123.0/24",
				// },
			},
			EniNetwork: CCEEniNetwork{
				Subnets: []string{},
			},
			Authentication: CCEAuthentication{
				Mode: "rbac",
				AuthenticatingProxy: CCEAuthenticatingProxy{
					Ca: "",
				},
			},
			BillingMode:          0,
			KubernetesSvcIPRange: "10.3.4.0/24",
			Tags: map[string]string{
				"cluster-key": "cluster-value",
			},
			KubeProxyMode: "iptables",
			PublicAccess:  true,
			PublicIP: CCEClusterPublicIP{
				CreateEIP: true,
				Eip: CCEEip{
					Iptype: "5_bgp",
					Bandwidth: CCEEipBandwidth{
						ChargeMode: "traffic",
						Size:       1,
						ShareType:  "PER",
					},
				},
			},
			NatGateway: CCENatGateway{
				Enabled: false,
				SNatRuleEIP: CCEEip{
					Iptype: "5_bgp",
					Bandwidth: CCEEipBandwidth{
						ChargeMode: "traffic",
						Size:       3,
						ShareType:  "PER",
					},
				},
				ExistingEIPID: "",
			},
			ExtendParam: CCEClusterExtendParam{
				ClusterAZ:         "cn-north-1a",
				ClusterExternalIP: "EIP-ADDR",
				PeriodType:        "month",
				PeriodNum:         0,
				IsAutoRenew:       "false",
				IsAutoPay:         "false",
			},
			NodePools: []CCENodePool{
				{
					Name: "nodepool-1",
					Type: "vm",
					ID:   "NODE_ID-aaa-bbb-ccc",
					NodeTemplate: CCENodeTemplate{
						Flavor:        "t6.large.2",
						AvailableZone: "cn-north-1a",
						SSHKey:        "SSH_KEY",
						RootVolume: CCENodeVolume{
							Size: 40,
							Type: "SSD",
						},
						DataVolumes: []CCENodeVolume{
							{
								Size: 100,
								Type: "SSD",
							},
						},
						BillingMode:     0,
						OperatingSystem: "EulerOS 2.9",
						PublicIP: CCENodePublicIP{
							Count: 1,
							Eip: CCEEip{
								Iptype: "5_bgp",
								Bandwidth: CCEEipBandwidth{
									ChargeMode: "traffic",
									Size:       1,
									ShareType:  "PER",
								},
							},
						},
						Runtime: "containerd",
						ExtendParam: CCENodeExtendParam{
							PeriodType:  "month",
							PeriodNum:   1,
							IsAutoRenew: "false",
						},
					},
					InitialNodeCount: 1,
					Autoscaling: CCENodePoolNodeAutoscaling{
						Enable:                false,
						MinNodeCount:          0,
						MaxNodeCount:          0,
						ScaleDownCooldownTime: 0,
						Priority:              0,
					},
					PodSecurityGroups: []string{},
					CustomSecurityGroups: []string{
						"SECURITY_GROUP_ID",
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
