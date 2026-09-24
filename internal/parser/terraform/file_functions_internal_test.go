package terraform

import "testing"

// The warning that tells a user how to enable a file function must cover
// exactly the functions that are gated; a gated function missing from the
// list would get the generic "not supported" message instead.
func TestFileFunctionNamesMatchGatedFunctions(t *testing.T) {
	gated := evalScope{rootDir: t.TempDir(), allowFiles: true}.fileFunctions()
	for name := range gated {
		if !fileFunctionNames[name] {
			t.Errorf("%s is gated but missing from fileFunctionNames", name)
		}
	}
	for name := range fileFunctionNames {
		if _, ok := gated[name]; !ok {
			t.Errorf("%s is in fileFunctionNames but not gated", name)
		}
		if _, ok := terraformFunctions()[name]; ok {
			t.Errorf("%s is gated but also registered unconditionally", name)
		}
	}
}
