package huawei

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	clusterDeleteError = `{"status_code":404,"request_id":"abababab","error_code":"CCE_CM.0003","error_message":"Resource not found","encoded_authorization_message":""}`
	subnetDeleteError  = `{"status_code":404,"request_id":"abababab","error_code":"VPC.0202","error_message":"Query resource by id abab-53ed-49f9-99a7-aabbcc fail.the subnet could not be found.","encoded_authorization_message":""}`
	vpcDeleteError     = `{"status_code":409,"request_id":"abababab","error_code":"VPC.0120","error_message":"{\"NeutronError\":{\"message\":\"exroutes exists under this router, delete exroutes firstly.\",\"type\":\"RouterInUse\",\"detail\":\"\"}}","encoded_authorization_message":""}`
)

func Test_IsHuaweiError(t *testing.T) {
	assert := assert.New(t)
	assert.False(IsHuaweiError(fmt.Errorf("AAABBB")))
	assert.True(IsHuaweiError(fmt.Errorf(vpcDeleteError)))
}

func Test_NewHuaweiError(t *testing.T) {
	assert := assert.New(t)
	vpcErr, err := NewHuaweiError(fmt.Errorf(vpcDeleteError))
	assert.Nil(err)
	if t.Failed() {
		return
	}
	fmt.Printf("%v\n", vpcErr.MarshalIndent())
	fmt.Printf("%v\n", vpcErr.String())
	clusterError, err := NewHuaweiError(fmt.Errorf(clusterDeleteError))
	assert.Nil(err)
	if t.Failed() {
		return
	}
	fmt.Printf("%v\n", clusterError.MarshalIndent())
	fmt.Printf("%v\n", clusterError.String())
	subnetError, err := NewHuaweiError(fmt.Errorf(subnetDeleteError))
	assert.Nil(err)
	if t.Failed() {
		return
	}
	fmt.Printf("%v\n", subnetError.MarshalIndent())
	fmt.Printf("%v\n", subnetError.String())
}
