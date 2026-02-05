package indexer

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	scip "github.com/sourcegraph/scip/bindings/go/scip"
	"google.golang.org/protobuf/proto"
)

func testdataDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata")
}

func indexTestdata(t *testing.T) *scip.Index {
	t.Helper()
	dir := testdataDir()
	tmpFile := filepath.Join(t.TempDir(), "index.scip")
	opts := Options{
		SourceDirs:  []string{dir},
		OutputFile:  tmpFile,
		ProjectRoot: dir,
	}
	if err := Run(opts); err != nil {
		t.Fatalf("indexing failed: %v", err)
	}
	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("reading index: %v", err)
	}
	var index scip.Index
	if err := proto.Unmarshal(data, &index); err != nil {
		t.Fatalf("unmarshaling index: %v", err)
	}
	return &index
}

func TestIndex_produces_documents(t *testing.T) {
	index := indexTestdata(t)
	if len(index.Documents) == 0 {
		t.Fatal("expected at least one document")
	}
	foundFiles := make(map[string]bool)
	for _, doc := range index.Documents {
		foundFiles[doc.RelativePath] = true
	}
	for _, expected := range []string{"MyClass.cls", "Greeter.cls", "Callable.cls", "Season.cls", "Worker.cls", "Outer.cls", "AccountTrigger.trigger", "CustomObjectUser.cls"} {
		if !foundFiles[expected] {
			t.Errorf("expected document for %s, got files: %v", expected, foundFiles)
		}
	}
}

func TestIndex_class_definitions_have_correct_ranges(t *testing.T) {
	index := indexTestdata(t)
	doc := findDoc(t, index, "MyClass.cls")
	defOccs := definitionOccurrences(doc)
	found := false
	for _, occ := range defOccs {
		if strings.HasSuffix(occ.Symbol, "MyClass#") {
			found = true
			if len(occ.Range) != 3 {
				t.Errorf("expected 3-element range, got %v", occ.Range)
				continue
			}
			// Line 0 (0-based), class name starts at some column
			if occ.Range[0] != 0 {
				t.Errorf("expected line 0, got %d", occ.Range[0])
			}
			break
		}
	}
	if !found {
		t.Error("no definition occurrence found for MyClass")
	}
}

func TestIndex_method_definitions_exist(t *testing.T) {
	index := indexTestdata(t)
	doc := findDoc(t, index, "MyClass.cls")
	defOccs := definitionOccurrences(doc)
	expectedMethods := []string{"getName().", "increment().", "getCount()."}
	for _, suffix := range expectedMethods {
		found := false
		for _, occ := range defOccs {
			if strings.HasSuffix(occ.Symbol, suffix) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("no definition found for method %s", suffix)
		}
	}
}

func TestIndex_field_definitions_exist(t *testing.T) {
	index := indexTestdata(t)
	doc := findDoc(t, index, "MyClass.cls")
	defOccs := definitionOccurrences(doc)
	expectedFields := []string{"name.", "count."}
	for _, suffix := range expectedFields {
		found := false
		for _, occ := range defOccs {
			if strings.HasSuffix(occ.Symbol, suffix) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("no definition found for field %s", suffix)
		}
	}
}

func TestIndex_interface_definition_exists(t *testing.T) {
	index := indexTestdata(t)
	doc := findDoc(t, index, "Callable.cls")
	defOccs := definitionOccurrences(doc)
	found := false
	for _, occ := range defOccs {
		if strings.HasSuffix(occ.Symbol, "Callable#") {
			found = true
			break
		}
	}
	if !found {
		t.Error("no definition found for Callable interface")
	}
}

func TestIndex_enum_definition_exists(t *testing.T) {
	index := indexTestdata(t)
	doc := findDoc(t, index, "Season.cls")
	defOccs := definitionOccurrences(doc)
	found := false
	for _, occ := range defOccs {
		if strings.HasSuffix(occ.Symbol, "Season#") {
			found = true
			break
		}
	}
	if !found {
		t.Error("no definition found for Season enum")
	}
}

func TestIndex_trigger_definition_exists(t *testing.T) {
	index := indexTestdata(t)
	doc := findDoc(t, index, "AccountTrigger.trigger")
	defOccs := definitionOccurrences(doc)
	found := false
	for _, occ := range defOccs {
		if strings.Contains(occ.Symbol, "AccountTrigger") {
			found = true
			break
		}
	}
	if !found {
		t.Error("no definition found for AccountTrigger")
	}
}

func TestIndex_nested_class_definition_exists(t *testing.T) {
	index := indexTestdata(t)
	doc := findDoc(t, index, "Outer.cls")
	defOccs := definitionOccurrences(doc)
	foundOuter := false
	foundInner := false
	for _, occ := range defOccs {
		if strings.HasSuffix(occ.Symbol, "Outer#") {
			foundOuter = true
		}
		if strings.Contains(occ.Symbol, "Outer#") && strings.HasSuffix(occ.Symbol, "Inner#") {
			foundInner = true
		}
	}
	if !foundOuter {
		t.Error("no definition found for Outer class")
	}
	if !foundInner {
		t.Error("no definition found for Outer.Inner nested class")
	}
}

func TestIndex_cross_file_references_exist(t *testing.T) {
	index := indexTestdata(t)
	doc := findDoc(t, index, "Greeter.cls")
	refOccs := referenceOccurrences(doc)
	found := false
	for _, occ := range refOccs {
		if strings.Contains(occ.Symbol, "MyClass") {
			found = true
			break
		}
	}
	if !found {
		t.Error("no cross-file reference to MyClass found in Greeter.cls")
	}
}

func TestIndex_constructor_references_exist(t *testing.T) {
	index := indexTestdata(t)
	doc := findDoc(t, index, "Worker.cls")
	refOccs := referenceOccurrences(doc)
	found := false
	for _, occ := range refOccs {
		if strings.Contains(occ.Symbol, "MyClass") {
			found = true
			break
		}
	}
	if !found {
		t.Error("no constructor reference to MyClass found in Worker.cls")
	}
}

func TestIndex_symbol_strings_are_well_formed(t *testing.T) {
	index := indexTestdata(t)
	for _, doc := range index.Documents {
		for _, occ := range doc.Occurrences {
			if occ.Symbol == "" {
				t.Errorf("empty symbol in %s at %v", doc.RelativePath, occ.Range)
				continue
			}
			if !strings.HasPrefix(occ.Symbol, "scip-apex ") && !strings.HasPrefix(occ.Symbol, "local ") {
				t.Errorf("unexpected symbol format in %s: %s", doc.RelativePath, occ.Symbol)
			}
		}
	}
}

func TestIndex_symbol_information_has_kinds(t *testing.T) {
	index := indexTestdata(t)
	for _, doc := range index.Documents {
		for _, si := range doc.Symbols {
			if si.Kind == scip.SymbolInformation_UnspecifiedKind {
				t.Errorf("unspecified kind for symbol %s in %s", si.Symbol, doc.RelativePath)
			}
		}
	}
}

func TestIndex_metadata_is_set(t *testing.T) {
	index := indexTestdata(t)
	if index.Metadata == nil {
		t.Fatal("metadata is nil")
	}
	if index.Metadata.ToolInfo == nil {
		t.Fatal("tool info is nil")
	}
	if index.Metadata.ToolInfo.Name != "scip-apex" {
		t.Errorf("expected tool name scip-apex, got %s", index.Metadata.ToolInfo.Name)
	}
	if index.Metadata.ProjectRoot == "" {
		t.Error("project root is empty")
	}
}

func TestIndex_implements_relationship(t *testing.T) {
	index := indexTestdata(t)
	doc := findDoc(t, index, "Worker.cls")
	for _, si := range doc.Symbols {
		if strings.HasSuffix(si.Symbol, "Worker#") {
			found := false
			for _, rel := range si.Relationships {
				if strings.Contains(rel.Symbol, "Callable") && rel.IsImplementation {
					found = true
					break
				}
			}
			if !found {
				t.Error("Worker should have implements relationship with Callable")
			}
			return
		}
	}
	t.Error("no SymbolInformation found for Worker class")
}

func TestIndex_all_occurrences_have_symbol_information(t *testing.T) {
	index := indexTestdata(t)
	// Collect all SymbolInformation from documents and external symbols
	knownSymbols := make(map[string]bool)
	for _, doc := range index.Documents {
		for _, si := range doc.Symbols {
			knownSymbols[si.Symbol] = true
		}
	}
	for _, si := range index.ExternalSymbols {
		knownSymbols[si.Symbol] = true
	}
	for _, doc := range index.Documents {
		for _, occ := range doc.Occurrences {
			if !knownSymbols[occ.Symbol] {
				t.Errorf("occurrence in %s at %v for symbol %s has no matching SymbolInformation",
					doc.RelativePath, occ.Range, occ.Symbol)
			}
		}
	}
}

func TestIndex_custom_object_metadata_files_not_indexed(t *testing.T) {
	index := indexTestdata(t)
	for _, doc := range index.Documents {
		if strings.HasSuffix(doc.RelativePath, ".object-meta.xml") ||
			strings.HasSuffix(doc.RelativePath, ".field-meta.xml") ||
			strings.HasSuffix(doc.RelativePath, ".object") {
			t.Errorf("metadata XML file should not be indexed as a document: %s", doc.RelativePath)
		}
	}
}

func TestIndex_custom_object_reference_from_apex(t *testing.T) {
	index := indexTestdata(t)
	doc := findDoc(t, index, "CustomObjectUser.cls")
	refOccs := referenceOccurrences(doc)
	found := false
	for _, occ := range refOccs {
		if strings.Contains(occ.Symbol, "TestObject__c") {
			found = true
			break
		}
	}
	if !found {
		t.Error("no reference to TestObject__c found in CustomObjectUser.cls")
	}
}

func TestIndex_custom_field_reference_from_apex(t *testing.T) {
	index := indexTestdata(t)
	doc := findDoc(t, index, "CustomObjectUser.cls")
	refOccs := referenceOccurrences(doc)
	found := false
	for _, occ := range refOccs {
		if strings.Contains(occ.Symbol, "CustomField__c") {
			found = true
			break
		}
	}
	if !found {
		t.Error("no reference to CustomField__c found in CustomObjectUser.cls")
	}
}

func TestIndex_standard_object_field_reference_from_trigger(t *testing.T) {
	index := indexTestdata(t)
	doc := findDoc(t, index, "AccountTrigger.trigger")
	refOccs := referenceOccurrences(doc)
	found := false
	for _, occ := range refOccs {
		if strings.Contains(occ.Symbol, "Account#") && strings.HasSuffix(occ.Symbol, "Name.") {
			found = true
			break
		}
	}
	if !found {
		t.Error("no reference to Account.Name found in AccountTrigger.trigger")
	}
}

func TestIndex_trigger_sobject_type_reference_exists(t *testing.T) {
	index := indexTestdata(t)
	doc := findDoc(t, index, "AccountTrigger.trigger")
	refOccs := referenceOccurrences(doc)
	found := false
	for _, occ := range refOccs {
		if strings.HasSuffix(occ.Symbol, "Account#") {
			found = true
			break
		}
	}
	if !found {
		t.Error("no type reference to Account found in AccountTrigger.trigger")
	}
}

func TestIndex_foreach_type_reference_exists(t *testing.T) {
	index := indexTestdata(t)
	doc := findDoc(t, index, "AccountTrigger.trigger")
	refOccs := referenceOccurrences(doc)
	count := 0
	for _, occ := range refOccs {
		if strings.HasSuffix(occ.Symbol, "Account#") {
			count++
		}
	}
	if count < 2 {
		t.Errorf("expected at least 2 Account type references (trigger + for-each), got %d", count)
	}
}

func TestIndex_variable_declaration_type_reference_exists(t *testing.T) {
	index := indexTestdata(t)
	doc := findDoc(t, index, "CustomObjectUser.cls")
	refOccs := referenceOccurrences(doc)
	count := 0
	for _, occ := range refOccs {
		if strings.HasSuffix(occ.Symbol, "TestObject__c#") {
			count++
		}
	}
	if count < 2 {
		t.Errorf("expected at least 2 TestObject__c references (variable type + constructor), got %d", count)
	}
}

func findDoc(t *testing.T, index *scip.Index, name string) *scip.Document {
	t.Helper()
	for _, doc := range index.Documents {
		if doc.RelativePath == name {
			return doc
		}
	}
	t.Fatalf("document %s not found", name)
	return nil
}

func definitionOccurrences(doc *scip.Document) []*scip.Occurrence {
	var result []*scip.Occurrence
	for _, occ := range doc.Occurrences {
		if occ.SymbolRoles&int32(scip.SymbolRole_Definition) != 0 {
			result = append(result, occ)
		}
	}
	return result
}

func referenceOccurrences(doc *scip.Document) []*scip.Occurrence {
	var result []*scip.Occurrence
	for _, occ := range doc.Occurrences {
		if occ.SymbolRoles&int32(scip.SymbolRole_Definition) == 0 {
			result = append(result, occ)
		}
	}
	return result
}
