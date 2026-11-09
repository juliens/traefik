package http

import (
	"errors"
	"fmt"
	"strings"

	"github.com/traefik/traefik/v3/pkg/rules"
)

const (
	// PriorityModeLength computes the default priority of a router from the length of its rule.
	PriorityModeLength = "length"
	// PriorityModeSpecificity computes the default priority of a router from the specificity of its rule matchers.
	PriorityModeSpecificity = "specificity"
)

// Bit layout of a specificity priority, from the most to the least significant criterion.
// Each criterion is saturated to its width before being shifted, so that no value of a criterion can ever outweigh
// a more significant one, whatever the number of matchers in the rule.
// The host kind never exceeds hostKindExplicit, so the 31 bits fit the router priority on 32-bit platforms, below the
// maximum user-defined priority.
const (
	queryBits      = 4
	headerBits     = 4
	methodBits     = 1
	pathBits       = 10
	exactBits      = 1
	narrowHostBits = 1
	hostLenBits    = 8

	queryShift      = 0
	headerShift     = queryShift + queryBits
	methodShift     = headerShift + headerBits
	pathShift       = methodShift + methodBits
	exactShift      = pathShift + pathBits
	narrowHostShift = exactShift + exactBits
	hostLenShift    = narrowHostShift + narrowHostBits
	hostShift       = hostLenShift + hostLenBits
)

const (
	hostKindNone = iota
	hostKindWildcard
	hostKindExplicit
)

const hostMatcher = "Host"

// RulePriority returns the default priority of a router with the given rule, computed with the given priority mode.
func (s SyntaxParser) RulePriority(syntax, rule, mode string) (int, error) {
	switch mode {
	case "", PriorityModeLength:
		return GetRulePriority(rule), nil

	case PriorityModeSpecificity:
		// The v2 syntax matchers take values whose length does not reflect their specificity, such as several hosts.
		if syntax == "v2" {
			return 0, errors.New("the specificity priority mode requires the v3 rule syntax")
		}

		p, ok := s.parsers["v3"]
		if !ok {
			return 0, errors.New("no parser for the v3 rule syntax")
		}

		parse, err := p.parser.Parse(rule)
		if err != nil {
			return 0, fmt.Errorf("parsing rule %s: %w", rule, err)
		}

		buildTree, ok := parse.(rules.TreeBuilder)
		if !ok {
			return 0, errors.New("obtaining build tree")
		}

		return treeSpecificity(buildTree()).priority(), nil

	default:
		return 0, fmt.Errorf("unknown priority mode %q", mode)
	}
}

// specificity holds the criteria ordering the routers matching the same request, as defined by the Gateway API
// HTTPRoute specification, which is also how NGINX selects a location and how the Kubernetes Ingress specification
// orders paths: the matching host first, then a matching host not narrowed from a broader one, an exact path, the
// longest path, a method, and the largest numbers of header and query matches.
// The other matchers, such as ClientIP or the negated ones, select no location in these specifications: they restrict
// what a location serves, so they leave the specificity unchanged, and the routers they distinguish are ordered by
// their order.
type specificity struct {
	hostKind int
	// hostLength only orders wildcard hosts: the explicit hosts matching the same request are the same host.
	hostLength int
	// broadHost is set when the rule also requires a host broader than its matching host, such as the hostname of an
	// HTTPRoute narrowed by the hostname of the listener it is attached to: the Gateway API specification orders the
	// routes yielding the same listener and route hostname intersection by their own hostname.
	broadHost  bool
	exactPath  bool
	pathLength int
	method     bool
	headers    int
	queries    int
}

func (s specificity) priority() int {
	return s.hostKind<<hostShift |
		saturate(s.hostLength, hostLenBits)<<hostLenShift |
		boolToInt(!s.broadHost)<<narrowHostShift |
		boolToInt(s.exactPath)<<exactShift |
		saturate(s.pathLength, pathBits)<<pathShift |
		boolToInt(s.method)<<methodShift |
		saturate(s.headers, headerBits)<<headerShift |
		saturate(s.queries, queryBits)<<queryShift
}

// and combines the criteria of two matchers which must both match.
func (s specificity) and(o specificity) specificity {
	s.broadHost = s.broadHost || o.broadHost ||
		s.hostKind != hostKindNone && o.hostKind != hostKindNone && (s.hostKind != o.hostKind || s.hostLength != o.hostLength)

	if o.hostKind > s.hostKind || (o.hostKind == s.hostKind && o.hostLength > s.hostLength) {
		s.hostKind, s.hostLength = o.hostKind, o.hostLength
	}

	if o.exactPath && !s.exactPath || o.exactPath == s.exactPath && o.pathLength > s.pathLength {
		s.exactPath, s.pathLength = o.exactPath, o.pathLength
	}

	s.method = s.method || o.method
	s.headers += o.headers
	s.queries += o.queries

	return s
}

func treeSpecificity(tree *rules.Tree) specificity {
	switch tree.Matcher {
	case "and":
		return treeSpecificity(tree.RuleLeft).and(treeSpecificity(tree.RuleRight))

	case "or":
		// Only what both alternatives guarantee is specific: a request may match either of them.
		left, right := treeSpecificity(tree.RuleLeft), treeSpecificity(tree.RuleRight)
		if left.priority() <= right.priority() {
			return left
		}
		return right
	}

	if tree.Not {
		return specificity{}
	}

	value := ""
	if len(tree.Value) > 0 {
		value = tree.Value[0]
	}

	switch tree.Matcher {
	case hostMatcher:
		switch {
		case value == "*":
			return specificity{}
		case strings.HasPrefix(value, "*"):
			// The single-label (*.) and multi-label (**.) wildcards of the same suffix are as specific, so that the
			// providers precedence decides between them.
			return specificity{hostKind: hostKindWildcard, hostLength: len(strings.TrimLeft(value, "*"))}
		default:
			return specificity{hostKind: hostKindExplicit}
		}

	case "HostRegexp":
		return specificity{hostKind: hostKindWildcard, hostLength: len(value)}

	case "Path":
		return specificity{exactPath: true, pathLength: len(value)}

	case "PathPrefix", "PathRegexp":
		return specificity{pathLength: len(value)}

	case "Method":
		return specificity{method: true}

	case "Header", "HeaderRegexp":
		return specificity{headers: 1}

	case "Query", "QueryRegexp":
		return specificity{queries: 1}

	default:
		return specificity{}
	}
}

func saturate(v, bits int) int {
	return min(v, 1<<bits-1)
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
