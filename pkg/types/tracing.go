package types

// TracingVerbosity defines the verbosity of the tracing of a router.
type TracingVerbosity string

const (
	MinimalVerbosity  TracingVerbosity = "minimal"
	DetailedVerbosity TracingVerbosity = "detailed"
)

// Allows reports whether the given verbosity is allowed by this verbosity.
func (v TracingVerbosity) Allows(verbosity TracingVerbosity) bool {
	switch v {
	case DetailedVerbosity:
		return verbosity == DetailedVerbosity || verbosity == MinimalVerbosity
	default:
		return verbosity == MinimalVerbosity
	}
}
