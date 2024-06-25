package transform_test

import (
	"testing"

	"github.com/zeiss/go-transform"

	"github.com/stretchr/testify/require"
)

func TestStruct(t *testing.T) {
	trans := transform.NewTransformer()

	tests := []struct {
		name string
		in   interface{}
		out  interface{}
	}{
		{
			name: "nil",
			in:   nil,
			out:  nil,
		},
		{
			name: "empty",
			in:   struct{}{},
			out:  struct{}{},
		},
		{
			name: "string",
			in: struct {
				Name string `transform:"trim,lowercase"`
			}{
				Name: "  test  ",
			},
			out: struct {
				Name string `transform:"trim,lowercase"`
			}{
				Name: "test",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := trans.Transform(tt.name, &tt.in)
			require.NoError(t, err)
			require.Equal(t, tt.out, tt.in)
		})
	}
}
