//go:build !hxstrict

package hx

// strictReads makes a Get outside of an effect a panic instead of a warning. It
// is turned on by building with the tag hxstrict.
const strictReads = false
