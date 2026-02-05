package indexer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/octoberswimmer/aer/ast"
	"github.com/octoberswimmer/aer/resolution"
	_ "github.com/octoberswimmer/aer/stdlib"
	scip "github.com/sourcegraph/scip/bindings/go/scip"
	"google.golang.org/protobuf/proto"
)

const version = "0.1.0"

type Options struct {
	SourceDirs  []string
	OutputFile  string
	ProjectRoot string
	PackageDir  string
	Namespace   string
}

func Run(opts Options) error {
	cu, err := parseSourceDirs(opts.SourceDirs)
	if err != nil {
		return fmt.Errorf("parsing source files: %w", err)
	}
	if cu == nil {
		return fmt.Errorf("no Apex source files found")
	}

	objectCU := parseObjectFiles(opts.SourceDirs)
	if objectCU != nil {
		cu.Classes = append(cu.Classes, objectCU.Classes...)
	}

	builder := resolution.NewBuilder()
	if err := builder.AddStandardLibrary(); err != nil {
		return fmt.Errorf("loading standard library: %w", err)
	}
	if opts.Namespace != "" {
		builder.AddCompilationUnitInNamespace(opts.Namespace, cu)
	} else {
		builder.AddCompilationUnit(cu)
	}
	graph, _ := builder.Build()

	resolver := resolution.NewResolver(graph)
	if opts.Namespace != "" {
		resolver.SetDefaultNamespace(opts.Namespace)
	}
	binding := resolver.ResolveCompilationUnit(cu)

	index := buildIndex(graph, binding, opts.ProjectRoot)
	return writeIndex(index, opts.OutputFile)
}

func parseSourceDirs(dirs []string) (*ast.CompilationUnit, error) {
	combined := &ast.CompilationUnit{}
	found := false
	for _, dir := range dirs {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(path))
			if ext != ".cls" && ext != ".trigger" {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("reading %s: %w", path, err)
			}
			absPath, err := filepath.Abs(path)
			if err != nil {
				absPath = path
			}
			cu, errs := ast.ParseASTWithFilenameWithOptions(string(data), absPath, ast.ParseOptions{
				CaptureDocComments: true,
			})
			if len(errs) > 0 {
				fmt.Fprintf(os.Stderr, "warning: parse errors in %s: %v\n", path, errs)
			}
			if cu == nil {
				return nil
			}
			found = true
			setFilePaths(cu, absPath)
			combined.Classes = append(combined.Classes, cu.Classes...)
			combined.Interfaces = append(combined.Interfaces, cu.Interfaces...)
			combined.Enums = append(combined.Enums, cu.Enums...)
			combined.Triggers = append(combined.Triggers, cu.Triggers...)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	if !found {
		return nil, nil
	}
	ast.CanonicalizeAST(combined)
	return combined, nil
}

func buildIndex(graph *resolution.SymbolGraph, binding *resolution.BindingResult, projectRoot string) *scip.Index {
	db := newDocumentBuilder(graph, binding, projectRoot)
	db.buildDefinitions()
	db.buildReferences()

	docs, externalSymbols := db.documents()
	return &scip.Index{
		Metadata: &scip.Metadata{
			Version: scip.ProtocolVersion_UnspecifiedProtocolVersion,
			ToolInfo: &scip.ToolInfo{
				Name:    "scip-apex",
				Version: version,
			},
			ProjectRoot:          "file://" + projectRoot,
			TextDocumentEncoding: scip.TextEncoding_UTF8,
		},
		Documents:       docs,
		ExternalSymbols: externalSymbols,
	}
}

func setFilePaths(cu *ast.CompilationUnit, absPath string) {
	for _, cls := range cu.Classes {
		cls.FilePath = absPath
		setNestedFilePaths(cls, absPath)
	}
	for _, iface := range cu.Interfaces {
		iface.FilePath = absPath
	}
	for _, enum := range cu.Enums {
		enum.FilePath = absPath
	}
	for _, trigger := range cu.Triggers {
		trigger.FilePath = absPath
	}
}

func setNestedFilePaths(cls *ast.ClassDeclaration, filePath string) {
	for _, nested := range cls.Nested {
		nested.FilePath = filePath
		setNestedFilePaths(nested, filePath)
	}
	for _, iface := range cls.NestedInterfaces {
		iface.FilePath = filePath
	}
	for _, enum := range cls.NestedEnums {
		enum.FilePath = filePath
	}
}

func writeIndex(index *scip.Index, outputFile string) error {
	data, err := proto.MarshalOptions{Deterministic: true}.Marshal(index)
	if err != nil {
		return fmt.Errorf("marshaling index: %w", err)
	}
	if err := os.WriteFile(outputFile, data, 0644); err != nil {
		return fmt.Errorf("writing %s: %w", outputFile, err)
	}
	fmt.Fprintf(os.Stderr, "Wrote %s (%d bytes, %d documents)\n", outputFile, len(data), len(index.Documents))
	return nil
}
