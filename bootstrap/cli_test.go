package bootstrap

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type codedError struct{ code int }

func (e codedError) Error() string { return "coded" }
func (e codedError) ExitCode() int { return e.code }

func TestExitCode(t *testing.T) {
	assert.Equal(t, 1, ExitCode(errors.New("plain")))
	assert.Equal(t, 3, ExitCode(codedError{code: 3}))
	assert.Equal(t, 3, ExitCode(fmt.Errorf("wrapped: %w", codedError{code: 3})))
}
