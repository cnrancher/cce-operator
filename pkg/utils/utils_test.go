package utils

import (
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func Test_WaitForCompleteWithError(t *testing.T) {
	err1 := fmt.Errorf("example error")
	err := WaitForCompleteWithError(func() error {
		fmt.Println("run func 1 called")
		time.Sleep(time.Second)
		fmt.Println("run func 1 finished")
		return err1
	})
	assert.ErrorIs(t, err, err1)
	t.Log("err1:", err)

	err = WaitForCompleteWithError(func() error {
		fmt.Println("run func 2 called")
		time.Sleep(time.Second)
		return nil
	})
	assert.Nil(t, err)
	t.Log("err2:", err)

	err = WaitForCompleteWithError(func() error {
		fmt.Println("run func 3 called")
		time.Sleep(DefaultDuration * 2) // timeout test
		logrus.Infof("func3 finished")  // this message should not output
		return nil
	})
	assert.ErrorIs(t, err, ErrWaitForCompleteTimeout)
	t.Log("err3:", err)
	t.Log("Go routine num:", runtime.NumGoroutine())
}
