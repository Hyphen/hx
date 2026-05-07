package timing

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRecorderMeasureNilReceiver(t *testing.T) {
	expectedErr := errors.New("boom")
	called := false
	var recorder *Recorder

	err := recorder.Measure("phase", func() error {
		called = true
		return expectedErr
	})

	assert.True(t, called)
	assert.ErrorIs(t, err, expectedErr)
}
