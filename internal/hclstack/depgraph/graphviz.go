package depgraph

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/c3xdev/c3x/internal/hclstack/options"
)

// WriteDot is used to emit a GraphViz compatible definition
// for a directed graph. It can be used to dump a .dot file.
// This is a similar implementation to terraform's digraph https://github.com/hashicorp/terraform/blob/master/digraph/graphviz.go
// adding some styling to modules that are excluded from the execution in *-all commands
func WriteDot(w io.Writer, terragruntOptions *options.TerragruntOptions, modules []*TerraformModule) error {

	_, _ = w.Write([]byte("digraph {\n"))
	defer func() { _, _ = w.Write([]byte("}\n")) }()

	// all paths are relative to the TerragruntConfigPath
	prefix := filepath.Dir(terragruntOptions.TerragruntConfigPath) + "/"

	for _, source := range modules {
		// apply a different coloring for excluded nodes
		style := ""
		if source.FlagExcluded {
			style = "[color=red]"
		}

		nodeLine := fmt.Sprintf("\t\"%s\" %s;\n",
			strings.TrimPrefix(source.Path, prefix), style)

		_, _ = w.Write([]byte(nodeLine))
		for _, target := range source.Dependencies {
			line := fmt.Sprintf("\t\"%s\" -> \"%s\";\n",
				strings.TrimPrefix(source.Path, prefix),
				strings.TrimPrefix(target.Path, prefix),
			)
			_, _ = w.Write([]byte(line))
		}
	}

	return nil
}
