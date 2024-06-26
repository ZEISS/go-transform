package transform_test

import (
	"fmt"
	"log"
	"testing"

	"github.com/zeiss/go-transform"

	"github.com/stretchr/testify/require"
)

func ExampleTransform() {
	type example struct {
		Name string `transform:"trim,lowercase"`
	}

	t := transform.NewTransformer()
	e := example{Name: "  John Doe  "}

	if err := t.Transform(&e); err != nil {
		log.Fatal(err)
	}

	fmt.Println(e.Name)
	// Output: john doe
}

func BenchmarkStruct(b *testing.B) {
	trans := transform.NewTransformer()

	type testStruct struct {
		Name string `transform:"trim"`
	}

	for i := 0; i < b.N; i++ {
		err := trans.Transform(&testStruct{Name: "  test  "})
		require.NoError(b, err)
	}
}

func TestNewTransformer(t *testing.T) {
	test := transform.NewTransformer()
	require.NotNil(t, test)
}

func TestStruct(t *testing.T) {
	trans := transform.NewTransformer()

	type testStruct struct {
		Name    string  `transform:"trim,lowercase"`
		NamePtr *string `transform:"trim,lowercase"`
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
				Name:    "  TEST  ",
				NamePtr: &[]string{"  TEST  "}[0],
			},
			out: &testStruct{
				Name:    "test",
				NamePtr: &[]string{"test"}[0],
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := trans.Transform(tt.in)
			require.NoError(t, err)
			require.Equal(t, tt.out, tt.in)
		})
	}
}

func TestStructLowercase(t *testing.T) {
	trans := transform.NewTransformer()

	type testStruct struct {
		Name string `transform:"lowercase"`
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
				Name: "TEST",
			},
			out: &testStruct{
				Name: "test",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := trans.Transform(tt.in)
			require.NoError(t, err)
			require.Equal(t, tt.out, tt.in)
		})
	}
}

func TestStructTrimRight(t *testing.T) {
	trans := transform.NewTransformer()

	type testStruct struct {
		Name string `transform:"rtrim"`
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
				Name: "test   ",
			},
			out: &testStruct{
				Name: "test",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := trans.Transform(tt.in)
			require.NoError(t, err)
			require.Equal(t, tt.out, tt.in)
		})
	}
}

func TestStructTrimLeft(t *testing.T) {
	trans := transform.NewTransformer()

	type testStruct struct {
		Name string `transform:"ltrim"`
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
				Name: "   test",
			},
			out: &testStruct{
				Name: "test",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := trans.Transform(tt.in)
			require.NoError(t, err)
			require.Equal(t, tt.out, tt.in)
		})
	}
}

func TestStructUppercase(t *testing.T) {
	trans := transform.NewTransformer()

	type testStruct struct {
		Name string `transform:"uppercase"`
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
				Name: "test",
			},
			out: &testStruct{
				Name: "TEST",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := trans.Transform(tt.in)
			require.NoError(t, err)
			require.Equal(t, tt.out, tt.in)
		})
	}
}
