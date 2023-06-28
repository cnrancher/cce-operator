package utils

import (
	"encoding/json"
	"errors"
	"math/rand"
	"strings"
	"time"
)

const (
	DefaultTimeout  = time.Second * 5
	DefaultDuration = time.Second * 5
)

var (
	ErrWaitForCompleteTimeout = errors.New("wait for complete timeout")
)

func PrintObject(a any) string {
	b, _ := json.MarshalIndent(a, "", "    ")
	return string(b)
}

func Parse(ref string) (namespace string, name string) {
	parts := strings.SplitN(ref, ":", 2)
	if len(parts) == 1 {
		return "", parts[0]
	}
	return parts[0], parts[1]
}

func WaitForCompleteWithError(f func() error) error {
	errCh := make(chan error)
	go func() {
		errCh <- f()
	}()

	select {
	case err := <-errCh:
		return err
	case <-time.After(DefaultTimeout):
		return ErrWaitForCompleteTimeout
	}
}

func RandomString(length int) string {
	rand.Seed(time.Now().UnixNano())
	chars := []byte("abcdefghijklmnopqrstuvwxyz0123456789")
	var b strings.Builder
	for i := 0; i < length; i++ {
		b.WriteByte(chars[rand.Intn(len(chars))])
	}

	return b.String()
}
