package ptr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPtrInt(t *testing.T) {
	testCases := []struct {
		name  string
		value int
	}{
		{"positive", 42},
		{"zero", 0},
		{"negative", -17},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ptrVal := Ptr(tc.value)
			require.NotNil(t, ptrVal, "Returned pointer should not be nil")
			require.Equal(t, tc.value, *ptrVal, "Dereferenced pointer should equal original value")
		})
	}
}

func TestPtrString(t *testing.T) {
	testCases := []struct {
		name  string
		value string
	}{
		{"non-empty", "hello world"},
		{"empty", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ptrVal := Ptr(tc.value)
			require.NotNil(t, ptrVal)
			require.Equal(t, tc.value, *ptrVal)
		})
	}
}

type testStruct struct {
	A int
	B string
}

func TestPtrStruct(t *testing.T) {
	testCases := []struct {
		name  string
		value testStruct
	}{
		{
			name: "simple",
			value: testStruct{
				A: 10,
				B: "foo",
			},
		},
		{
			name: "zero",
			value: testStruct{
				A: 0,
				B: "",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ptrVal := Ptr(tc.value)
			require.NotNil(t, ptrVal)
			require.Equal(t, tc.value, *ptrVal)
		})
	}
}

func TestPtrSlice(t *testing.T) {
	testCases := []struct {
		name  string
		value []string
	}{
		{
			name:  "non-empty",
			value: []string{"a", "b", "c"},
		},
		{
			name:  "empty",
			value: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ptrVal := Ptr(tc.value)
			require.NotNil(t, ptrVal)
			require.Equal(t, tc.value, *ptrVal)
		})
	}
}
