package mapper

import "testing"

func TestValueMatcher_Matches__RuleEqual(t *testing.T) {
	tests := []struct {
		matcherValue   any
		valueToCompare any
		want           bool
	}{
		{"hello", "hello", true},
		{"hello", "Hello", false},
		{"hello", "world", false},

		{1, 1, true},
		{1, 2, false},

		{1.0, 1.0, true},
		{1.0, 2.0, false},

		{true, true, true},
		{true, false, false},

		{nil, nil, true},
		{nil, 1, false},
		{nil, "hello", false},
		{"hello", nil, false},
	}

	for _, tt := range tests {
		matcher := &ValueMatcher{Rule: MatchingRuleEqual, Value: tt.matcherValue}

		matched := matcher.Matches(tt.valueToCompare)
		if matched != tt.want {
			t.Errorf("%v == %v,  ValueMatcher.Matches() = %v, want %v", tt.matcherValue, tt.valueToCompare, matched, tt.want)
		}
	}
}

func TestValueMatcher_Matches__RuleContains(t *testing.T) {
	tests := []struct {
		name           string
		value          any
		wantToContains any
		want           bool
	}{
		{
			name:           "string",
			value:          "hello world",
			wantToContains: "world",
			want:           true,
		},
		{
			name:           "string NOT contains number",
			value:          "answer 42",
			wantToContains: 42,
			want:           false,
		},
		{
			name:           "string contains rune",
			value:          "abc",
			wantToContains: 'a',
			want:           true,
		},
		{
			name:           "string NOT contains different string",
			value:          "golang",
			wantToContains: "java",
			want:           false,
		},
		{
			name:           "empty string NOT contains string",
			value:          "",
			wantToContains: "x",
			want:           false,
		},

		{
			name:           "int slice",
			value:          []int{1, 2, 3},
			wantToContains: 2,
			want:           true,
		},
		{
			name:           "slice contains subslice",
			value:          [][]int{{1, 2}, {3, 4}},
			wantToContains: []int{3, 4},
			want:           true,
		},
		{
			name:           "any slice contains int ",
			value:          []interface{}{"1", 2, 3},
			wantToContains: 1,
			want:           false,
		},
		{
			name:           "empty slice NOT contains string",
			value:          []string{},
			wantToContains: "test",
			want:           false,
		},

		{
			name:           "map ignored",
			value:          map[int]string{1: "a"},
			wantToContains: "a",
			want:           false,
		},
		{
			name:           "int ignored",
			value:          100,
			wantToContains: 100,
			want:           false,
		},
		{
			name:           "channel ignored",
			value:          make(chan int),
			wantToContains: nil,
			want:           false,
		},

		{
			name:           "nil slice",
			value:          nil,
			wantToContains: "test",
			want:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher := &ValueMatcher{Rule: MatchingRuleContains, Value: tt.wantToContains}

			if matched := matcher.Matches(tt.value); matched != tt.want {
				t.Errorf("'%v' contains '%v',  ValueMatcher.Matches() = %v, want %v", tt.value, tt.wantToContains, matched, tt.want)
			}
		})
	}
}
