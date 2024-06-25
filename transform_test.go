package transform_test

import (
	"testing"

	"github.com/zeiss/go-transform"

	"github.com/stretchr/testify/require"
)

func TestStruct(t *testing.T) {
	trans := transform.NewTransformer()

	type testStruct struct {
		Name string `transform:"trim"`
	}

	tests := []struct {
		name string
		in   *testStruct
		out  *testStruct
	}{
		{
			name: "nil",
			in:   nil,
			out:  nil,
		},
		{
			name: "empty",
			in:   &testStruct{},
			out:  &testStruct{},
		},
		{
			name: "string",
			in: &testStruct{
				Name: "  test  ",
			},
			out: &testStruct{
				Name: "test",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := trans.Transform(tt.name, tt.in)
			require.NoError(t, err)
			require.Equal(t, tt.out, tt.in)
		})
	}
}
