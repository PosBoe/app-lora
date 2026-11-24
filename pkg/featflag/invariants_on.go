//go:build invariants

package featflag

// Invariants is true when the binary is built with the invariants tag. Code
// uses it to panic on a broken invariant in development. A production build
// returns a graceful error instead.
const Invariants = true
