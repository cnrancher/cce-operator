## cnrancher/cce-operator

[![Build Status](https://drone-pandaria.cnrancher.com/api/badges/cnrancher/cce-operator/status.svg?ref=refs/heads/main)](https://drone-pandaria.cnrancher.com/cnrancher/cce-operator)
[![Docker Pulls](https://img.shields.io/docker/pulls/cnrancher/cce-operator.svg)](https://store.docker.com/community/images/cnrancher/cce-operator)
[![Go Report Card](https://goreportcard.com/badge/github.com/cnrancher/cce-operator)](https://goreportcard.com/report/github.com/cnrancher/cce-operator)

Kubernetes controller for managing [Huawei Cloud Container Engine](https://www.huaweicloud.com/product/cce.html) (CCE) in Rancher.

### Develop

The easiest way to debug and develop the operator is to replace the default operator on a running Rancher instance with your local one (see [eks-operator](https://github.com/rancher/eks-operator#develop)).

You can also build and debug CCE Operator without Rancher by following these steps:

1. Setup a kubernetes cluster and configure the `KUBECONFIG` file.

    ```console
    $ export KUBECONFIG="$HOME/.kube/config"
    ```

1. Create a `Opaque` type secret (huawei cloud credential) in namespace `cattle-global-data`.

    ```console
    $ kubectl create namespace cattle-global-data
    ```

    ```yaml
    apiVersion: v1
    kind: Secret
    type: Opaque
    metadata:
        name: "cc-test-cce" # Modify the secret name if needed.
        namespace: cattle-global-data
    data:
        huaweicredentialConfig-accessKey: "[base64_encoded_access_key]"
        huaweicredentialConfig-secretKey: "[base64_encoded_secret_key]"
        huaweicredentialConfig-projectID: "[base64_encoded_project_id]"
        huaweicredentialConfig-regionID: "[base64_encoded_region_id]"
    ```

1. Clone this project and build the executable binary.

    ```console
    $ git clone https://github.com/cnrancher/cce-operator.git && cd cce-operator
    $ go generate
    $ go build .
    ```

1. Apply the CRD config file.

    ```console
    $ kubectl apply -f ./charts/cce-operator-crd/templates/crds.yaml
    ```

1. Run the operator and then apply the example configs to create/import cluster.

    ```console
    $ ./cce-operator --debug
    ```

    Modify the YAML configs in [examples](./examples/) manually such as `huaweiCredentialSecret`, `regionID`, `hostNetwork`, `nodeTemplate.sshKey` etc.

    Launch another terminal for applying the YAML config files.

    ```console
    $ kubectl apply -f ./examples/create-example.yaml
    ```

### Documents

The Simplified Chinese documentation of CRD parameters is in the [examples/docs](./examples/docs) directory.

### Versions

The version correspondence between CCE Operator and Rancher is as follows.

| cce-operator | Rancher  |
|:------------:|:--------:|
| `v0.1.x`     | N/A      |
| `v0.2.x`     | `v2.6.x` |
| `v0.3.x`     | `v2.7.x` |
| `v0.4.x`     | `v2.8.x` |

### LICENSE

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
