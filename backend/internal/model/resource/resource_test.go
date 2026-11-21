package resource

import (
	"reflect"
	"testing"
)

func TestParseDeviceIDs(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		want    []uint64
		wantErr bool
	}{
		{
			name: "string ids from merchant frontend",
			raw:  `["3726401","3726802","3727303"]`,
			want: []uint64{3726401, 3726802, 3727303},
		},
		{
			name: "numeric ids",
			raw:  `[3726401,3726802]`,
			want: []uint64{3726401, 3726802},
		},
		{
			name: "mixed ids",
			raw:  `["3726401",3726802]`,
			want: []uint64{3726401, 3726802},
		},
		{name: "empty", raw: "", want: nil},
		{name: "null", raw: "null", want: nil},
		{name: "empty array", raw: "[]", want: []uint64{}},
		{name: "invalid id", raw: `["abc"]`, wantErr: true},
		{name: "invalid json", raw: `["1",`, wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseDeviceIDs(tc.raw)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("ParseDeviceIDs(%q) expected error, got %v", tc.raw, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseDeviceIDs(%q) error: %v", tc.raw, err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("ParseDeviceIDs(%q) = %v, want %v", tc.raw, got, tc.want)
			}
		})
	}
}
