package cce_test

import (
	"fmt"
	"testing"

	"github.com/cnrancher/cce-operator/pkg/huawei/cce"
	"github.com/cnrancher/cce-operator/pkg/utils"
)

func Test_ListAddonInstances(t *testing.T) {
	if client == nil {
		return
	}
	res, err := cce.ListAddonInstances(client, "", "")
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("%s\n", utils.PrintObject(res))
}

func Test_CreateAddonInstance(t *testing.T) {
	if client == nil {
		return
	}
	res, err := cce.CreateAddonInstance(client)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("%s\n", utils.PrintObject(res))
}
