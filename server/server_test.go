package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServer_Init(t *testing.T) {
	s, err := New()
	assert.NoError(t, err)
	assert.Equal(t, ":5432", s.address)
}

func TestServer_WithAddress(t *testing.T) {
	s, err := New(WithAddress("localhost:2021"))
	assert.NoError(t, err)
	assert.Equal(t, "localhost:2021", s.address)
}

func TestServer_WithPort(t *testing.T) {
	s, err := New(WithPort(1986))
	assert.NoError(t, err)
	assert.Equal(t, ":1986", s.address)
}
