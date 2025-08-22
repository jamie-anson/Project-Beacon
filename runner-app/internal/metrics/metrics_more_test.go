package metrics

import "testing"

func TestFmtInt_Cases(t *testing.T) {
	cases := map[int]string{
		0:    "0",
		7:    "7",
		42:   "42",
		999:  "999",
		-1:   "-1",
		-123: "-123",
	}
	for in, want := range cases {
		got := fmtInt(in)
		if got != want {
			t.Fatalf("fmtInt(%d)=%q want %q", in, got, want)
		}
	}
}
