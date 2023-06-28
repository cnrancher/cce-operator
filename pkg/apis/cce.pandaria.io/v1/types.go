/*
Copyright 2023 [Rancher Labs, Inc](https://rancher.com).

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type CCEClusterConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CCEClusterConfigSpec   `json:"spec"`
	Status CCEClusterConfigStatus `json:"status"`
}

// CCEClusterConfigSpec is the spec for a CCEClusterConfig resource
type CCEClusterConfigSpec struct {
	CredentialSecret     string            `json:"credentialSecret"`
	Category             string            `json:"category,omitempty"` // 集群类别: CCE
	RegionID             string            `json:"regionID,omitempty"`
	ClusterID            string            `json:"clusterID,omitempty"` // ClusterID only used for import cluster
	Imported             bool              `json:"imported,omitempty"`
	Name                 string            `json:"name" norman:"noupdate"`
	Labels               map[string]string `json:"labels,omitempty"`
	Type                 string            `json:"type"`
	Flavor               string            `json:"flavor" norman:"noupdate"`
	Version              string            `json:"version" norman:"noupdate"`
	Description          string            `json:"description,omitempty" norman:"noupdate"`
	Ipv6Enable           bool              `json:"ipv6Enable,omitempty"`
	HostNetwork          HostNetwork       `json:"hostNetwork"`
	ContainerNetwork     ContainerNetwork  `json:"containerNetwork"`
	EniNetwork           EniNetwork        `json:"eniNetwork,omitempty"`
	Authentication       Authentication    `json:"authentication,omitempty"`
	BillingMode          int32             `json:"clusterBillingMode,omitempty" norman:"noupdate"`
	KubernetesSvcIPRange string            `json:"kubernetesSvcIPRange,omitempty" norman:"noupdate"`
	Tags                 map[string]string `json:"tags"`
	KubeProxyMode        string            `json:"kubeProxyMode,omitempty"`
	NodeConfigs          []NodeConfig      `json:"nodeConfigs,omitempty"`
}

type CCEClusterConfigStatus struct {
	Phase          string `json:"phase"`
	FailureMessage string `json:"failureMessage"`

	ClusterID        string           `json:"clusterID"`
	HostNetwork      HostNetwork      `json:"hostNetwork"`
	ContainerNetwork ContainerNetwork `json:"containerNetwork"`
	ClusterEIPID     string           `json:"clusterEIPID,omitempty"`
	ELBID            string           `json:"elbID,omitempty"`
	PoolID           string           `json:"poolID,omitempty"`
	VipSubnetID      string           `json:"vipSubnetID,omitempty"`
	NodeConfigs      []NodeConfig     `json:"nodeConfigs,omitempty"`
}

type HostNetwork struct {
	VpcID         string `json:"vpcID,omitempty"`
	SubnetID      string `json:"subnetID,omitempty"`
	SecurityGroup string `json:"securityGroup,omitempty"`
}

type ContainerNetwork struct {
	Mode string `json:"mode"`
	CIDR string `json:"cidr"`
	// CIDRs []string `json:"cidrs"` // 10.0.0.0/12~19, 172.16.0.0/16~19, 192.168.0.0/16~19
}

type EniNetwork struct {
	Subnets []string `json:"subnets"`
}

type Authentication struct {
	Mode                string              `json:"mode"`
	AuthenticatingProxy AuthenticatingProxy `json:"authenticatingProxy"`
}

type AuthenticatingProxy struct {
	Ca         string `json:"ca,omitempty"`
	Cert       string `json:"cert,omitempty"`
	PrivateKey string `json:"privateKey,omitempty"`
}

type NodeConfig struct {
	Name            string            `json:"name,omitempty"`
	NodeID          string            `json:"nodeID,omitempty"`
	Flavor          string            `json:"flavor"`
	AvailableZone   string            `json:"availableZone"`
	SSHKey          string            `json:"sshKey"`
	RootVolume      Volume            `json:"rootVolume"`
	DataVolumes     []Volume          `json:"dataVolumes"`
	BillingMode     int32             `json:"billingMode"`
	OperatingSystem string            `json:"operatingSystem"`
	PublicIP        PublicIP          `json:"publicIP"`
	ExtendParam     ExtendParam       `json:"extendParam"`
	Labels          map[string]string `json:"labels"`
	Count           int32             `json:"count"` // 批量创建节点时的数量
}

type Volume struct {
	Size int32  `json:"size"`
	Type string `json:"type"`
}

type Bandwidth struct {
	ChargeMode string `json:"chargeMode,omitempty"`
	Size       int32  `json:"size,omitempty"`
	ShareType  string `json:"shareType,omitempty"`
}

type Eip struct {
	Iptype    string    `json:"ipType,omitempty"`
	Bandwidth Bandwidth `json:"bandwidth,omitempty"`
}

type PublicIP struct {
	Ids   []string `json:"ids,omitempty"`
	Count int32    `json:"count,omitempty"`
	Eip   Eip      `json:"eip,omitempty"`
}

type ExtendParam struct {
	BMSPeriodType  string `json:"periodType,omitempty"`
	BMSPeriodNum   int32  `json:"periodNum,omitempty"`
	BMSIsAutoRenew string `json:"isAutoRenew,omitempty"`
}
