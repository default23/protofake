package mapper

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/gobwas/glob"
)

// MatchingRule is the rule, defining how the value should be matched.
type MatchingRule string

const (
	// MatchingRuleEqual is the rule, defining the value should be strictly equal, case-sensitive.
	MatchingRuleEqual MatchingRule = "equal"
	// MatchingRuleEqualIgnoreCase is the rule, defining the value should be strictly equal, case-insensitive.
	MatchingRuleEqualIgnoreCase MatchingRule = "iequal"
	// MatchingRuleContains is the rule, defining the value should contain the given value.
	MatchingRuleContains MatchingRule = "contains"
	// MatchingRuleRegex is the rule, defining the value should match the given regex.
	MatchingRuleRegex MatchingRule = "regex"
	// MatchingRuleGlob is the rule, defining the value should match the given glob pattern.
	// Glob is a wildcard pattern, where * is a wildcard for any character, and ? is a wildcard for one character.
	MatchingRuleGlob MatchingRule = "glob"
)

// ValueMatcher is the matching rule for the Value.
type ValueMatcher struct {
	Rule MatchingRule `json:"rule"`
	// Value is the value to match
	Value any `json:"value"`
}

// NewValueMatcher creates a new ValueMatcher.
func NewValueMatcher(rule MatchingRule, val any) (*ValueMatcher, error) {
	if !rule.IsValid() {
		return nil, fmt.Errorf("invalid matching rule: %s", rule)
	}

	str, ok := val.(string)
	if ok { // for the string any rule is valid
		switch rule { //nolint:exhaustive
		case MatchingRuleRegex:
			_, err := regexp.Compile(str)
			if err != nil {
				return nil, fmt.Errorf("invalid regex pattern: %w", err)
			}

		case MatchingRuleGlob:
			_, err := glob.Compile(str)
			if err != nil {
				return nil, fmt.Errorf("invalid glob pattern: %w", err)
			}
		}

		return &ValueMatcher{Rule: rule, Value: val}, nil
	}

	if rule != MatchingRuleEqual && rule != MatchingRuleContains {
		return nil, fmt.Errorf("the rule %s is only valid for string values", rule)
	}

	// validate for primitives only
	switch t := val.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
	case float32, float64:
	case bool:
	case nil:
	default:
		return nil, fmt.Errorf("the rule %s is only valid for primitive values, got %T", rule, t)
	}

	return &ValueMatcher{Rule: rule, Value: val}, nil
}

// Matches checks if the given value satisfies the rule.
func (m *ValueMatcher) Matches(val any) bool {
	switch m.Rule {
	case MatchingRuleEqual:
		return m.deepEqual(val)
	case MatchingRuleEqualIgnoreCase:
		a, b := reflect.ValueOf(m.Value), reflect.ValueOf(val)
		if a.Kind() == reflect.String && b.Kind() == reflect.String {
			return strings.EqualFold(a.String(), b.String())
		}

		return m.deepEqual(val)
	case MatchingRuleContains:
		return m.contains(val)
	case MatchingRuleRegex:
		mval, ok := m.Value.(string)
		if !ok {
			return false
		}
		vval, ok := val.(string)
		if !ok {
			return false
		}
		re, err := regexp.Compile(mval)
		if err != nil {
			return false
		}

		return re.MatchString(vval)
	case MatchingRuleGlob:
		globVal, ok := m.Value.(string)
		if !ok {
			return false
		}
		vval, ok := val.(string)
		if !ok {
			return false
		}
		g, err := glob.Compile(globVal)
		if err != nil {
			return false
		}

		return g.Match(vval)
	}

	return false
}

// IsValid checks if the rule is valid matching rule type.
func (mr MatchingRule) IsValid() bool {
	switch mr {
	case MatchingRuleEqual,
		MatchingRuleEqualIgnoreCase,
		MatchingRuleContains,
		MatchingRuleRegex,
		MatchingRuleGlob:
		return true
	}

	return false
}
