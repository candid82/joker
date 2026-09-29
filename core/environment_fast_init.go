//go:build !gen_code && !packed_core
// +build !gen_code,!packed_core

package core

func (env *Env) ReferCoreToUser() {
	// Nothing need be done; it's already "baked in" in the fast-startup version.
}
