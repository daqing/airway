package repo

import (
	"testing"
)

func TestBuildCondQuery(t *testing.T) {
	tests := []struct {
		Pairs     []KVPair
		Start     int
		Separator Separator

		ExpectedCondQuery string
		ExpectedValues    []any
		ExpectedDollar    int
	}{
		{
			[]KVPair{
				KV("foo", "bar"),
			},
			0,
			and_sep,

			"foo = $1",
			[]any{"bar"},
			2,
		},
		{
			[]KVPair{
				KV("foo", "bar"),
				&OrQuery{
					[]KVPair{
						KV("A", 1),
						KV("B", 2),
					},
				},
			},
			0,
			and_sep,

			"foo = $1 AND A = $2 OR B = $3",
			[]any{"bar", 1, 2},
			4,
		},
		{
			[]KVPair{
				KV("foo", "bar"),
				KV("age", 32),
			},
			0,
			or_sep,

			"foo = $1 OR age = $2",
			[]any{"bar", 32},
			3,
		},
		{
			[]KVPair{
				&OrQuery{
					[]KVPair{
						KV("name", "david"),
						KV("age", 30),
					},
				},
				&OrQuery{
					[]KVPair{
						KV("city", "beijing"),
						KV("code", "100000"),
					},
				},

				KV("count", 9),
			},
			0,
			and_sep,

			"name = $1 OR age = $2 AND city = $3 OR code = $4 AND count = $5",
			[]any{"david", 30, "beijing", "100000", 9},
			6,
		},
	}

	for _, test := range tests {
		condQuery, vals, dollor := buildCondQuery(test.Pairs, test.Start, test.Separator)

		if condQuery != test.ExpectedCondQuery {
			t.Errorf("[Cond Query] expected %v, got %v", test.ExpectedCondQuery, condQuery)
		}

		if len(vals) != len(test.ExpectedValues) {
			t.Errorf("[values] expected %v, got %v", test.ExpectedValues, vals)
		}

		if dollor != test.ExpectedDollar {
			t.Errorf("[dollar] expected %d, got %d", test.ExpectedDollar, dollor)
		}
	}
}
