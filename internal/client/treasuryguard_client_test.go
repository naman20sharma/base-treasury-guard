package client

import (
	"math/big"
	"testing"
)

func TestAsUint64Parsing(t *testing.T) {
	okCases := []struct {
		name string
		val  any
		want uint64
	}{
		{"uint64", uint64(42), 42},
		{"uint32", uint32(7), 7},
		{"uint", uint(9), 9},
		{"int64", int64(11), 11},
		{"bigInt", big.NewInt(99), 99},
	}
	for _, tc := range okCases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := asUint64(tc.val)
			if !ok {
				t.Fatalf("expected ok for %s", tc.name)
			}
			if got != tc.want {
				t.Fatalf("got %d want %d", got, tc.want)
			}
		})
	}

	var nilBig *big.Int
	bigToo := new(big.Int).Lsh(big.NewInt(1), 65)
	badCases := []struct {
		name string
		val  any
	}{
		{"negInt64", int64(-1)},
		{"nilBig", nilBig},
		{"bigTooLarge", bigToo},
		{"bigNegative", big.NewInt(-1)},
	}
	for _, tc := range badCases {
		t.Run(tc.name, func(t *testing.T) {
			if _, ok := asUint64(tc.val); ok {
				t.Fatalf("expected reject for %s", tc.name)
			}
		})
	}
}
