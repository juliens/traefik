package http

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRulePriority(t *testing.T) {
	parser, err := NewSyntaxParser()
	require.NoError(t, err)

	priority, err := parser.RulePriority("v3", `Host("foo.com") && PathPrefix("/")`, "")
	require.NoError(t, err)
	assert.Equal(t, len(`Host("foo.com") && PathPrefix("/")`), priority)

	priority, err = parser.RulePriority("v3", `Host("foo.com") && PathPrefix("/")`, PriorityModeLength)
	require.NoError(t, err)
	assert.Equal(t, len(`Host("foo.com") && PathPrefix("/")`), priority)

	_, err = parser.RulePriority("v2", "Host(`foo.com`)", PriorityModeSpecificity)
	require.Error(t, err)

	_, err = parser.RulePriority("v3", `Host("foo.com")`, "unknown")
	require.Error(t, err)

	_, err = parser.RulePriority("v3", `Host("foo.com"`, PriorityModeSpecificity)
	require.Error(t, err)
}

func TestRulePriority_specificityOrder(t *testing.T) {
	manyHeaders := func(n int) string {
		var rules []string
		for i := range n {
			rules = append(rules, fmt.Sprintf("Header(%q, %q)", fmt.Sprintf("h%d", i), "v"))
		}
		return strings.Join(rules, " && ")
	}

	testCases := []struct {
		desc   string
		higher string
		lower  string
	}{
		{
			desc:   "Explicit host before wildcard host",
			higher: `Host("foo.example.com") && PathPrefix("/")`,
			lower:  `Host("*.example.com") && Path("/a/very/long/path") && Method("GET") && ` + manyHeaders(20),
		},
		{
			desc:   "Explicit host before wildcard host regexp",
			higher: `Host("foo.example.com") && PathPrefix("/")`,
			lower:  `HostRegexp("^[a-z0-9-\\.]+\\.example\\.com$") && Path("/x")`,
		},
		{
			desc:   "Wildcard host before no host",
			higher: `Host("*.example.com") && PathPrefix("/")`,
			lower:  `Path("/a/very/long/path") && Method("GET") && ` + manyHeaders(20),
		},
		{
			desc:   "Longer wildcard host before shorter wildcard host",
			higher: `Host("*.a.example.com") && PathPrefix("/")`,
			lower:  `Host("*.example.com") && Path("/x")`,
		},
		{
			desc:   "Explicit hosts are not ordered by their length",
			higher: `Host("a.com") && PathPrefix("/ab")`,
			lower:  `Host("a-much-longer-host.com") && PathPrefix("/a")`,
		},
		{
			desc:   "Exact path before longer prefix path",
			higher: `Host("foo.com") && Path("/a")`,
			lower:  `Host("foo.com") && PathPrefix("/a/very/long/path") && Method("GET") && ` + manyHeaders(20),
		},
		{
			desc:   "Element-wise prefix path is a prefix path",
			higher: `Host("foo.com") && Path("/a")`,
			lower:  `Host("foo.com") && (Path("/a") || PathPrefix("/a/"))`,
		},
		{
			desc:   "Longer prefix path before shorter prefix path with a method and headers",
			higher: `Host("foo.com") && (Path("/abc") || PathPrefix("/abc/"))`,
			lower:  `Host("foo.com") && (Path("/a") || PathPrefix("/a/")) && Method("GET") && ` + manyHeaders(20),
		},
		{
			desc:   "Method before any number of headers",
			higher: `Host("foo.com") && PathPrefix("/o") && Method("GET")`,
			lower:  `Host("foo.com") && PathPrefix("/o") && ` + manyHeaders(1000),
		},
		{
			desc:   "More headers before fewer headers",
			higher: `Host("foo.com") && PathPrefix("/h") && ` + manyHeaders(2),
			lower:  `Host("foo.com") && PathPrefix("/h") && ` + manyHeaders(1),
		},
		{
			desc:   "Headers before any number of query params",
			higher: `Host("foo.com") && PathPrefix("/q") && Header("h", "v")`,
			lower:  `Host("foo.com") && PathPrefix("/q") && Query("q1", "v") && Query("q2", "v") && Query("q3", "v")`,
		},
		{
			desc:   "Additional matcher before the same rule without it",
			higher: `Host("foo.com") && PathPrefix("/") && ClientIP("10.0.0.0/8")`,
			lower:  `Host("foo.com") && PathPrefix("/")`,
		},
		{
			desc:   "Negated host does not designate a host",
			higher: `Host("*.example.com") && PathPrefix("/")`,
			lower:  `!Host("foo.example.com") && PathPrefix("/a/very/long/path")`,
		},
		{
			desc:   "Alternative between an explicit and a wildcard host is a wildcard host",
			higher: `Host("bar.example.com") && PathPrefix("/")`,
			lower:  `(Host("foo.example.com") || Host("*.example.com")) && Path("/x")`,
		},
	}

	parser, err := NewSyntaxParser()
	require.NoError(t, err)

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			higher, err := parser.RulePriority("v3", test.higher, PriorityModeSpecificity)
			require.NoError(t, err)

			lower, err := parser.RulePriority("v3", test.lower, PriorityModeSpecificity)
			require.NoError(t, err)

			assert.Greater(t, higher, lower)
		})
	}
}

func TestRulePriority_specificityBounds(t *testing.T) {
	parser, err := NewSyntaxParser()
	require.NoError(t, err)

	var headers, queries []string
	for i := range 100 {
		headers = append(headers, fmt.Sprintf("Header(%q, %q)", fmt.Sprintf("h%d", i), "v"))
		queries = append(queries, fmt.Sprintf("Query(%q, %q)", fmt.Sprintf("q%d", i), "v"))
	}

	rule := fmt.Sprintf(`Host("*.%s.com") && Path("/%s") && Method("GET") && %s && %s && ClientIP("10.0.0.0/8")`,
		strings.Repeat("a", 300), strings.Repeat("b", 2000), strings.Join(headers, " && "), strings.Join(queries, " && "))

	priority, err := parser.RulePriority("v3", rule, PriorityModeSpecificity)
	require.NoError(t, err)

	// The router priorities above MaxInt32-1000 are reserved to the internal routers on 32-bit platforms.
	assert.Less(t, priority, math.MaxInt32-1000)

	explicit, err := parser.RulePriority("v3", `Host("foo.com")`, PriorityModeSpecificity)
	require.NoError(t, err)
	assert.Less(t, priority, explicit)
}
