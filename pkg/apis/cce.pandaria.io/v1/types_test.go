package v1

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/rancher/wrangler/pkg/yaml"
)

func Test_CCEClusterConfig_Create(t *testing.T) {
	c := CCEClusterConfig{
		Spec: CCEClusterConfigSpec{
			CredentialSecret: "cattle-global-data:cc-test",
			Category:         "CCE",
			ClusterID:        "aaa-bbb-ccc",
			Imported:         false,
			Name:             "cce-create-1",
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
			BillingMode:          int32(0),
			KubernetesSvcIPRange: "10.3.4.0/24",
			Tags: map[string]string{
				"cluster-key": "cluster-value",
			},
			KubeProxyMode: "",
			NodePools: []CCENodePool{
				{
					Name: "nodepool-1",
					Type: "vm",
					ID:   "NODE_ID-aaa-bbb-ccc",
					NodeTemplate: CCENodeTemplate{
						Flavor:        "t6.large.2",
						AvailableZone: "cn-north-1a",
						SSHKey:        "SSH_KEY",
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
						Runtime: "containerd",
						ExtendParam: ExtendParam{
							PeriodType:  "month",
							PeriodNum:   1,
							IsAutoRenew: "false",
						},
						Count: int32(1),
					},
					InitialNodeCount: 1,
					Autoscaling: NodePoolNodeAutoscaling{
						Enable:                false,
						MinNodeCount:          1,
						MaxNodeCount:          1,
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

func Test_CCEClusterConfig_Create2(t *testing.T) {
	// convert create-example yaml spec to json
	d, err := os.ReadFile("../../../../examples/create-example.yaml")
	if err != nil {
		t.Error(err)
		return
	}
	var tmp any
	err = yaml.Unmarshal(d, &tmp)
	if err != nil {
		t.Error(err)
		return
	}
	config := tmp.(map[string]any)
	data, err := json.MarshalIndent(config["spec"], "", "    ")
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("%v\n", string(data))
}
