package huawei

import (
	"bytes"
	"encoding/json"
)

type HuaweiError struct {
	StatusCode   int32  `json:"status_code,omitempty"`
	RequestID    string `json:"request_id,omitempty"`
	ErrorCode    string `json:"error_code,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

func NewHuaweiError(err error) (*HuaweiError, error) {
	var errMsg string
	if err != nil {
		errMsg = err.Error()
	}
	d := json.NewDecoder(bytes.NewBufferString(errMsg))
	huaweiError := &HuaweiError{}
	err = d.Decode(huaweiError)
	return huaweiError, err
}

func IsHuaweiError(err error) bool {
	d := json.NewDecoder(bytes.NewBufferString(err.Error()))
	h := &HuaweiError{}
	if err := d.Decode(h); err != nil {
		return false
	}
	return true
}

func (e *HuaweiError) String() string {
	d, _ := json.Marshal(e)
	return string(d)
}

func (e *HuaweiError) MarshalIndent() string {
	d, _ := json.MarshalIndent(e, "", "  ")
	return string(d)
}
