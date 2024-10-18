package main

import (
	"fmt"
	"os"

	v12 "github.com/cnrancher/cce-operator/pkg/apis/cce.pandaria.io/v1"
	controllergen "github.com/rancher/wrangler/v2/pkg/controller-gen"
	"github.com/rancher/wrangler/v2/pkg/controller-gen/args"
	"github.com/rancher/wrangler/v2/pkg/crd"
	"github.com/rancher/wrangler/v2/pkg/yaml"
	corev1 "k8s.io/api/core/v1"
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
			"": {
				Types: []interface{}{
					corev1.Pod{},
					corev1.Node{},
					corev1.Secret{},
				},
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
	if err := os.WriteFile(filename, []byte(data), 0600); err != nil {
		return err
	}

	return nil
}
