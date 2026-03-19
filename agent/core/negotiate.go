package core

// ContentSupport declares which media types a provider handles natively.
type ContentSupport struct {
	NativeTypes map[MediaType]bool
}

// Supports returns true if the given media type is natively supported.
func (cs ContentSupport) Supports(mt MediaType) bool {
	return cs.NativeTypes[mt]
}

// ContentNegotiator is an optional interface for provider adapters
// to declare native file support.
type ContentNegotiator interface {
	ContentSupport() ContentSupport
}
