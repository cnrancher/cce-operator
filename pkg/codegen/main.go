package main

import (
	"fmt"
	"os"

	v12 "github.com/cnrancher/cce-operator/pkg/apis/cce.pandaria.io/v1"
	_ "github.com/rancher/wrangler-api/pkg/generated/controllers/apiextensions.k8s.io"
	controllergen "github.com/rancher/wrangler/pkg/controller-gen"
	"github.com/rancher/wrangler/pkg/controller-gen/args"
	"github.com/rancher/wrangler/pkg/crd"
	"github.com/rancher/wrangler/pkg/yaml"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func main() {
	os.Unsetenv("GOPATH")

	controllergen.Run(args.Options{
		OutputPackage: "github.com/cnrancher/cce-operator/pkg/generated",
		Boilerplate:   "pkg/codegen/boilerplate.go.txt",
		Groups: map[string]args.Group{
			"cce.pandaria.io": {
				Types: []interface{}{
					"./pkg/apis/cce.pandaria.io/v1",
				},
				GenerateTypes: true,
			},
			// Optionally you can use wrangler-api project which
			// has a lot of common kubernetes APIs already generated.
			// In this controller we will use wrangler-api for apps api group
			"": {
				Types: []interface{}{
					v1.Pod{},
					v1.Node{},
					v1.Secret{},
				},
				InformersPackage: "k8s.io/client-go/informers",
				ClientSetPackage: "k8s.io/client-go/kubernetes",
				ListersPackage:   "k8s.io/client-go/listers",
			},
		},
	})

	cceClusterConfig := newCRD(&v12.CCEClusterConfig{}, func(c crd.CRD) crd.CRD {
		c.ShortNames = []string{"ccecc"}
		return c
	})

	obj, err := cceClusterConfig.ToCustomResourceDefinition()
	if err != nil {
		panic(err)
	}

	obj.(*unstructured.Unstructured).SetAnnotations(map[string]string{
		"helm.sh/resource-policy": "keep",
	})

	cceCCYaml, err := yaml.Export(obj)
	if err != nil {
		panic(err)
	}

	if err := saveCRDYaml("cce-operator-crd", string(cceCCYaml)); err != nil {
		panic(err)
	}

	// fmt.Printf("obj yaml: %s", cceCCYaml)
}

func newCRD(obj interface{}, customize func(crd.CRD) crd.CRD) crd.CRD {
	crd := crd.CRD{
		GVK: schema.GroupVersionKind{
			Group:   "cce.pandaria.io",
			Version: "v1",
		},
		Status:       true,
		SchemaObject: obj,
	}
	if customize != nil {
		crd = customize(crd)
	}
	return crd
}

func saveCRDYaml(name, data string) error {
	filename := fmt.Sprintf("./charts/%s/templates/crds.yaml", name)
	if err := os.WriteFile(filename, []byte(data), 0644); err != nil {
		return err
	}

	return nil
}
