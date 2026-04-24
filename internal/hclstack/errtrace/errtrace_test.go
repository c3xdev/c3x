package errtrace

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithStackTrace(t *testing.T) {
	err := WithStackTrace(errors.New("test error"))
	assert.NotNil(t, err)
	assert.Equal(t, "test error", err.Error())
}

func TestWithStackTrace_Nil(t *testing.T) {
	err := WithStackTrace(nil)
	assert.Nil(t, err)
}

func TestWithStackTrace_Unwrap(t *testing.T) {
	original := errors.New("original")
	wrapped := WithStackTrace(original)
	assert.True(t, errors.Is(wrapped, original))
}

func TestWithStackTraceAndPrefix(t *testing.T) {
	err := WithStackTraceAndPrefix(errors.New("failed"), "reading file %s", "test.txt")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "reading file test.txt")
	assert.Contains(t, err.Error(), "failed")
}

func TestWithStackTraceAndPrefix_Nil(t *testing.T) {
	err := WithStackTraceAndPrefix(nil, "prefix")
	assert.Nil(t, err)
}

func TestErrorf(t *testing.T) {
	err := Errorf("error %d: %s", 42, "test")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "error 42: test")
}

func TestNew(t *testing.T) {
	original := errors.New("original error")
	wrapped := New(original)
	assert.NotNil(t, wrapped)
	assert.Equal(t, "original error", wrapped.Error())
}

func TestRecover_Nil(t *testing.T) {
	err := Recover(nil)
	assert.Nil(t, err)
}

func TestRecover_Error(t *testing.T) {
	err := Recover(errors.New("panic error"))
	assert.NotNil(t, err)
	assert.Equal(t, "panic error", err.Error())
}

func TestRecover_String(t *testing.T) {
	err := Recover("string panic")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "string panic")
}

func TestIsError(t *testing.T) {
	err1 := fmt.Errorf("error 1")
	err2 := fmt.Errorf("error 2")
	assert.True(t, IsError(err1, err2)) // same type (*errors.errorString)
}

func TestPrintErrorWithEncoder(t *testing.T) {
	var encoded interface{}
	encoder := func(v interface{}) error {
		encoded = v
		return nil
	}
	PrintErrorWithEncoder(errors.New("test"), encoder)
	assert.Equal(t, "test", encoded)
}
