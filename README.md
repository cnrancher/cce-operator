## cnrancher/cce-operator

[![Build Status](https://drone-pandaria.cnrancher.com/api/badges/cnrancher/cce-operator/status.svg?ref=refs/heads/main)](https://drone-pandaria.cnrancher.com/cnrancher/cce-operator)
[![Docker Pulls](https://img.shields.io/docker/pulls/cnrancher/cce-operator.svg)](https://store.docker.com/community/images/cnrancher/cce-operator)
[![Go Report Card](https://goreportcard.com/badge/github.com/cnrancher/cce-operator)](https://goreportcard.com/report/github.com/cnrancher/cce-operator)

Kubernetes controller for managing [Huawei Cloud Container Engine](https://www.huaweicloud.com/product/cce.html) (CCE) in Rancher.

### Usage

You can build and debug `cce-operator` outside of Rancher by following these steps:

1. Setup a kubernetes cluster and configure the `KUBECONFIG` file:

    ```console
    $ export KUBECONFIG="$HOME/.kube/config"
    ```

1. Create a `Opaque` type secret (huawei cloud credential) in namespace `cattle-global-data`:

    ```console
    $ kubectl create namespace cattle-global-data
    ```

    ```yaml
    apiVersion: v1
    kind: Secret
    type: Opaque
    metadata:
        name: "[secret-name]"
        namespace: cattle-global-data
    data:
        huaweicredentialConfig-accessKey: "[base64 encoded access key]"
        huaweicredentialConfig-secretKey: "[base64 encoded secret key]"
        huaweicredentialConfig-projectID: "[base64 encoded project id]"
        huaweicredentialConfig-regionID: "[base64 encoded region id]"
    ```

1. Build the operator executable file:

    ```console
    $ git clone https://github.com/cnrancher/cce-operator.git && cd cce-operator
    $ go generate
    $ go build .
    ```

1. Apply the CRD:

    ```console
    $ kubectl apply -f ./charts/cce-operator-crd/templates/crds.yaml
    ```

1. Run the operator and create/import CCE cluster:

    ```console
    $ ./cce-operator --debug
    ```

    Modify the `CredentialSecret`, `hostNetwork`, `sshKey` and other configurations in `examples/create-example.yaml` and `examples/import-example.yaml`.

    Launch another terminal for applying the yaml config files:

    ```console
    $ kubectl apply -f ./examples/create-example.yaml
    ```

### Examples

You can find CRD definition examples and parameter documentation in the [examples/](./examples/) directory.

### LICENSE

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
