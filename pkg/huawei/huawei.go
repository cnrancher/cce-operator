package huawei

import (
	"bytes"
	"encoding/json"
)

type Error struct {
	StatusCode   int32  `json:"status_code,omitempty"`
	RequestID    string `json:"request_id,omitempty"`
	ErrorCode    string `json:"error_code,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

func NewHuaweiError(err error) (*Error, error) {
	var errMsg string
	if err != nil {
		errMsg = err.Error()
	}
	d := json.NewDecoder(bytes.NewBufferString(errMsg))
	huaweiError := &Error{}
	err = d.Decode(huaweiError)
	return huaweiError, err
}

func IsHuaweiError(err error) bool {
	d := json.NewDecoder(bytes.NewBufferString(err.Error()))
	h := &Error{}
	if err := d.Decode(h); err != nil {
		return false
	}
	return true
}

func (e *Error) String() string {
	d, _ := json.Marshal(e)
	return string(d)
}

func (e *Error) MarshalIndent() string {
	d, _ := json.MarshalIndent(e, "", "  ")
	return string(d)
}
