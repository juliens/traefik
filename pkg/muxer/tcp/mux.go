package tcp

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"regexp"
	"sort"
	"strings"

	"github.com/traefik/traefik/v2/pkg/ip"
	"github.com/traefik/traefik/v2/pkg/log"
	"github.com/traefik/traefik/v2/pkg/rules"
	"github.com/traefik/traefik/v2/pkg/tcp"
	"github.com/traefik/traefik/v2/pkg/types"
	"github.com/vulcand/predicate"
)

var funcs = map[string]func(*matchersTree, ...string) error{
	"HostSNI":  hostSNI,
	"ClientIP": clientIP,
}

// ParseHostSNI extracts the HostSNIs declared in a rule.
// This is a first naive implementation used in TCP routing.
func ParseHostSNI(rule string) ([]string, error) {
	var matchers []string
	for matcher := range funcs {
		matchers = append(matchers, matcher)
	}

	parser, err := rules.NewParser(matchers)
	if err != nil {
		return nil, err
	}

	parse, err := parser.Parse(rule)
	if err != nil {
		return nil, err
	}

	buildTree, ok := parse.(rules.TreeBuilder)
	if !ok {
		return nil, errors.New("cannot parse")
	}

	return buildTree().ParseMatchers([]string{"HostSNI"}), nil
}

// ConnData contains TCP connection metadata.
type ConnData struct {
	serverName string
	remoteIP   string
}

// NewConnData builds a connData struct from the given parameters.
func NewConnData(serverName string, conn tcp.WriteCloser) (ConnData, error) {
	ip, _, err := net.SplitHostPort(conn.RemoteAddr().String())
	if err != nil {
		return ConnData{}, fmt.Errorf("error while parsing remote address %q: %w", conn.RemoteAddr().String(), err)
	}

	// as per https://datatracker.ietf.org/doc/html/rfc6066:
	// > The hostname is represented as a byte string using ASCII encoding without a trailing dot.
	// so there is no need to trim a potential trailing dot
	serverName = types.CanonicalDomain(serverName)

	return ConnData{
		serverName: types.CanonicalDomain(serverName),
		remoteIP:   ip,
	}, nil
}

// Muxer defines a muxer that handles TCP routing with rules.
type Muxer struct {
	routes   []*route
	parser   predicate.Parser
	NotFound tcp.Handler
}

// NewMuxer returns a TCP muxer.
func NewMuxer() (*Muxer, error) {
	var matcherNames []string
	for matcherName := range funcs {
		matcherNames = append(matcherNames, matcherName)
	}

	parser, err := rules.NewParser(matcherNames)
	if err != nil {
		return nil, fmt.Errorf("error while creating rules parser: %w", err)
	}

	return &Muxer{parser: parser, NotFound: tcp.HandlerFunc(func(conn tcp.WriteCloser) {
		conn.Close()
	})}, nil
}

func (m Muxer) ServeTCP(conn tcp.WriteCloser) {
	eConn := &EnrichConn{WriteCloser: conn}
	h := m.Match(eConn)
	if h != nil {
		h.ServeTCP(eConn)
		return
	}
	m.NotFound.ServeTCP(eConn)
}

// Match returns the handler of the first route matching the connection metadata.
func (m Muxer) Match(meta *EnrichConn) tcp.Handler {
	for _, route := range m.routes {
		if route.matchers.match(meta) {
			return route.handler
		}
	}

	return nil
}

// AddRoute adds a new route, associated to the given handler, at the given
// priority, to the muxer.
func (m *Muxer) AddRoute(rule string, priority int, handler tcp.Handler, tls bool) error {
	// Special case for when the catchAll fallback is present.
	// When no user-defined priority is found, the lowest computable priority minus one is used,
	// in order to make the fallback the last to be evaluated.
	if priority == 0 && rule == "HostSNI(`*`)" {
		priority = -2
		if tls {
			priority = -1
		}
	}

	// Default value, which means the user has not set it, so we'll compute it.
	if priority == 0 {
		priority = len(rule)
	}

	parse, err := m.parser.Parse(rule)
	if err != nil {
		return fmt.Errorf("error while parsing rule %s: %w", rule, err)
	}

	buildTree, ok := parse.(rules.TreeBuilder)
	if !ok {
		return fmt.Errorf("error while parsing rule %s", rule)
	}

	var matchers matchersTree
	err = addRule(&matchers, buildTree())
	if err != nil {
		return err
	}

	var realMatchers *matchersTree
	realMatchers = &matchers
	if tls {
		var tlsMatcher matchersTree
		tlsMatcher.matcher = func(meta *EnrichConn) bool {
			return meta.IsTLS() && matchers.match(meta)
		}
		realMatchers = &tlsMatcher
	}

	newRoute := &route{
		handler:  handler,
		priority: priority,
		matchers: *realMatchers,
		rule:     fmt.Sprintf("Rule: %s Priority: %d TLS: %v", rule, priority, tls),
	}
	m.routes = append(m.routes, newRoute)

	sort.Sort(routes(m.routes))

	return nil
}

func addRule(tree *matchersTree, rule *rules.Tree) error {
	switch rule.Matcher {
	case "and", "or":
		tree.operator = rule.Matcher
		tree.left = &matchersTree{}
		err := addRule(tree.left, rule.RuleLeft)
		if err != nil {
			return err
		}

		tree.right = &matchersTree{}
		return addRule(tree.right, rule.RuleRight)
	default:
		err := rules.CheckRule(rule)
		if err != nil {
			return err
		}

		err = funcs[rule.Matcher](tree, rule.Value...)
		if err != nil {
			return err
		}

		if rule.Not {
			matcherFunc := tree.matcher
			tree.matcher = func(meta *EnrichConn) bool {
				return !matcherFunc(meta)
			}
		}
	}

	return nil
}

// HasRoutes returns whether the muxer has routes.
func (m *Muxer) HasRoutes() bool {
	return len(m.routes) > 0
}

// routes implements sort.Interface.
type routes []*route

// Len implements sort.Interface.
func (r routes) Len() int { return len(r) }

// Swap implements sort.Interface.
func (r routes) Swap(i, j int) { r[i], r[j] = r[j], r[i] }

// Less implements sort.Interface.
func (r routes) Less(i, j int) bool { return r[i].priority > r[j].priority }

// route holds the matchers to match TCP route,
// and the handler that will serve the connection.
type route struct {
	// The matchers tree structure reflecting the rule.
	matchers matchersTree
	// The handler responsible for handling the route.
	handler tcp.Handler

	// Used to disambiguate between two (or more) rules that would both match for a
	// given request.
	// Computed from the matching rule length, if not user-set.
	priority int

	rule string
}

func (r *route) String() string {
	return r.rule
}

// matcher is a matcher func used to match connection properties.
type matcher func(meta *EnrichConn) bool

// matchersTree represents the matchers tree structure.
type matchersTree struct {
	// If matcher is not nil, it means that this matcherTree is a leaf of the tree.
	// It is therefore mutually exclusive with left and right.
	matcher matcher
	// operator to combine the evaluation of left and right leaves.
	operator string
	// Mutually exclusive with matcher.
	left  *matchersTree
	right *matchersTree
}

func (m *matchersTree) match(meta *EnrichConn) bool {
	if m == nil {
		// This should never happen as it should have been detected during parsing.
		log.WithoutContext().Warnf("Rule matcher is nil")
		return false
	}

	if m.matcher != nil {
		return m.matcher(meta)
	}

	switch m.operator {
	case "or":
		return m.left.match(meta) || m.right.match(meta)
	case "and":
		return m.left.match(meta) && m.right.match(meta)
	default:
		// This should never happen as it should have been detected during parsing.
		log.WithoutContext().Warnf("Invalid rule operator %s", m.operator)
		return false
	}
}

func clientIP(tree *matchersTree, clientIPs ...string) error {
	checker, err := ip.NewChecker(clientIPs)
	if err != nil {
		return fmt.Errorf("could not initialize IP Checker for \"ClientIP\" matcher: %w", err)
	}

	tree.matcher = func(meta *EnrichConn) bool {
		if meta.GetRemoteIP() == "" {
			return false
		}

		ok, err := checker.Contains(meta.GetRemoteIP())
		if err != nil {
			log.WithoutContext().Warnf("\"ClientIP\" matcher: could not match remote address : %v", err)
			return false
		}
		if ok {
			return true
		}

		return false
	}

	return nil
}

var almostFQDN = regexp.MustCompile(`^[[:alnum:]\.-]+$`)

// hostSNI checks if the SNI Host of the connection match the matcher host.
func hostSNI(tree *matchersTree, hosts ...string) error {
	if len(hosts) == 0 {
		return fmt.Errorf("empty value for \"HostSNI\" matcher is not allowed")
	}

	for i, host := range hosts {
		// Special case to allow global wildcard
		if host == "*" {
			continue
		}

		if !almostFQDN.MatchString(host) {
			return fmt.Errorf("invalid value for \"HostSNI\" matcher, %q is not a valid hostname", host)
		}

		hosts[i] = strings.ToLower(host)
	}

	tree.matcher = func(meta *EnrichConn) bool {
		// Since a HostSNI(`*`) rule has been provided as catchAll for non-TLS TCP,
		// it allows matching with an empty serverName.
		// Which is why we make sure to take that case into account before before
		// checking meta.serverName.
		if hosts[0] == "*" {
			return true
		}

		if meta.ServerName() == "" {
			return false
		}

		for _, host := range hosts {
			if host == "*" {
				return true
			}

			if host == meta.ServerName() {
				return true
			}

			// trim trailing period in case of FQDN
			host = strings.TrimSuffix(host, ".")
			if host == meta.ServerName() {
				return true
			}
		}

		return false
	}

	return nil
}

type EnrichConn struct {
	tcp.WriteCloser
	calculated bool
	serverName string
	isTLS      bool
}

func (e *EnrichConn) GetRemoteIP() string {
	ip, _, _ := net.SplitHostPort(e.RemoteAddr().String())
	return ip
}

func (e *EnrichConn) calculate() {
	br := bufio.NewReader(e.WriteCloser)
	serverName, tls, peeked, err := clientHelloServerName(br)
	if err != nil {
		log.WithoutContext().Error(err)
	}

	e.serverName = serverName
	e.isTLS = tls
	e.WriteCloser = &Conn{
		Peeked:      []byte(peeked),
		WriteCloser: e.WriteCloser,
	}
}
func (e *EnrichConn) IsTLS() bool {
	if !e.calculated {
		e.calculate()
	}
	return e.isTLS
}

func (e *EnrichConn) ServerName() string {
	if !e.calculated {
		e.calculate()
	}
	return e.serverName
}

// Conn is a connection proxy that handles Peeked bytes.
type Conn struct {
	// Peeked are the bytes that have been read from Conn for the
	// purposes of route matching, but have not yet been consumed
	// by Read calls. It set to nil by Read when fully consumed.
	Peeked []byte

	// Conn is the underlying connection.
	// It can be type asserted against *net.TCPConn or other types
	// as needed. It should not be read from directly unless
	// Peeked is nil.
	tcp.WriteCloser
}

// Read reads bytes from the connection (using the buffer prior to actually reading).
func (c *Conn) Read(p []byte) (n int, err error) {
	if len(c.Peeked) > 0 {
		n = copy(p, c.Peeked)
		c.Peeked = c.Peeked[n:]
		if len(c.Peeked) == 0 {
			c.Peeked = nil
		}
		return n, nil
	}
	return c.WriteCloser.Read(p)
}

const defaultBufSize = 4096

// clientHelloServerName returns the SNI server name inside the TLS ClientHello,
// without consuming any bytes from br.
// On any error, the empty string is returned.
func clientHelloServerName(br *bufio.Reader) (string, bool, string, error) {
	hdr, err := br.Peek(1)
	if err != nil {
		var opErr *net.OpError
		if !errors.Is(err, io.EOF) && (!errors.As(err, &opErr) || opErr.Timeout()) {
			log.WithoutContext().Errorf("Error while Peeking first byte: %s", err)
		}

		return "", false, "", err
	}

	// No valid TLS record has a type of 0x80, however SSLv2 handshakes
	// start with a uint16 length where the MSB is set and the first record
	// is always < 256 bytes long. Therefore typ == 0x80 strongly suggests
	// an SSLv2 client.
	const recordTypeSSLv2 = 0x80
	const recordTypeHandshake = 0x16
	if hdr[0] != recordTypeHandshake {
		if hdr[0] == recordTypeSSLv2 {
			// we consider SSLv2 as TLS and it will be refused by real TLS handshake.
			return "", true, getPeeked(br), nil
		}
		return "", false, getPeeked(br), nil // Not TLS.
	}

	const recordHeaderLen = 5
	hdr, err = br.Peek(recordHeaderLen)
	if err != nil {
		log.Errorf("Error while Peeking hello: %s", err)
		return "", false, getPeeked(br), nil
	}

	recLen := int(hdr[3])<<8 | int(hdr[4]) // ignoring version in hdr[1:3]

	if recordHeaderLen+recLen > defaultBufSize {
		br = bufio.NewReaderSize(br, recordHeaderLen+recLen)
	}

	helloBytes, err := br.Peek(recordHeaderLen + recLen)
	if err != nil {
		log.Errorf("Error while Hello: %s", err)
		return "", true, getPeeked(br), nil
	}

	sni := ""
	server := tls.Server(sniSniffConn{r: bytes.NewReader(helloBytes)}, &tls.Config{
		GetConfigForClient: func(hello *tls.ClientHelloInfo) (*tls.Config, error) {
			sni = hello.ServerName
			return nil, nil
		},
	})
	_ = server.Handshake()

	return sni, true, getPeeked(br), nil
}

func getPeeked(br *bufio.Reader) string {
	peeked, err := br.Peek(br.Buffered())
	if err != nil {
		log.Errorf("Could not get anything: %s", err)
		return ""
	}
	return string(peeked)
}

// sniSniffConn is a net.Conn that reads from r, fails on Writes,
// and crashes otherwise.
type sniSniffConn struct {
	r        io.Reader
	net.Conn // nil; crash on any unexpected use
}

// Read reads from the underlying reader.
func (c sniSniffConn) Read(p []byte) (int, error) { return c.r.Read(p) }

// Write crashes all the time.
func (sniSniffConn) Write(p []byte) (int, error) { return 0, io.EOF }
