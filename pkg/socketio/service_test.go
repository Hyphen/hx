package socketio

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestService(t *testing.T) {
	t.Run("markConnected_closes_connectedCh_on_first_call", func(t *testing.T) {
		s := NewService()

		s.markConnected()

		select {
		case <-s.connectedCh:
			// connectedCh closed as expected
		default:
			t.Fatal("expected connectedCh to be closed after markConnected()")
		}
	})

	t.Run("markConnected_is_idempotent_across_repeated_calls", func(t *testing.T) {
		s := NewService()

		assert.NotPanics(t, func() {
			s.markConnected()
			s.markConnected()
			s.markConnected()
		}, "repeated markConnected calls must not panic on close-of-closed-channel")
	})
}
