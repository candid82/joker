//go:build gen_code
// +build gen_code

package core

// CompileAST is a no-op during code generation.
// The real implementation is in compile_fast_init.go.
func CompileAST(expr Expr) {
	// No-op during gen_code to avoid expensive compilation overhead
}
