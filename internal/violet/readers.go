package violet

import "bytes"

// newBytesReader is a tiny helper so seed.go doesn't need to import bytes
// directly (keeps seed.go focused on the embed contract).
func newBytesReader(b []byte) *bytes.Reader { return bytes.NewReader(b) }
