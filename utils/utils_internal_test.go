package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsPointer(t *testing.T) {
	testCases := []struct {
		name     string
		input    any
		expected bool
	}{
		{
			name:     "int value",
			input:    42,
			expected: false,
		},
		{
			name:     "int pointer",
			input:    new(int),
			expected: true,
		},
		{
			name:     "string value",
			input:    "hello",
			expected: false,
		},
		{
			name:     "string pointer",
			input:    new(string),
			expected: true,
		},
		{
			name:     "nil value",
			input:    nil,
			expected: false,
		},
		{
			name:     "struct value",
			input:    struct{}{},
			expected: false,
		},
		{
			name:     "struct pointer",
			input:    &struct{}{},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := IsPointer(tc.input)
			require.Equal(t, tc.expected, actual, "Test case: %s failed", tc.name)
		})
	}
}
