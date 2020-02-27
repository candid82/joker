// +build !slow_init

package core

func (env *Env) ReferCoreToUser() {
	// Nothing need be done; it's already "baked in" in the fast-startup version.
}
