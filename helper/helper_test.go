package helper

import (
	"github.com/magiconair/properties/assert"
	"testing"
)

func TestSnakeify(t *testing.T) {
	var tests = []struct {
		in       string
		expected string
	}{
		{"The     Agent", "the_agent"},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			out := Snakeify(tt.in)

			assert.Equal(t, out, tt.expected)
		})
	}
}

func TestKebabify(t *testing.T) {
	var tests = []struct {
		in       string
		expected string
	}{
		{"The     Agent", "the-agent"},
		{"not_an_ __url", "not-an-url"},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			out := Kebabify(tt.in)

			assert.Equal(t, out, tt.expected)
		})
	}
}

func TestPascalify(t *testing.T) {
	var tests = []struct {
		in       string
		expected string
	}{
		{"the     agent", "TheAgent"},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			out := Pascalify(tt.in)

			assert.Equal(t, out, tt.expected)
		})
	}
}
