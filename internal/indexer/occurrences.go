package indexer

import (
	"path/filepath"
	"strings"

	"github.com/octoberswimmer/aer/ast"
	"github.com/octoberswimmer/aer/resolution"
	scip "github.com/sourcegraph/scip/bindings/go/scip"
)

type documentBuilder struct {
	graph       *resolution.SymbolGraph
	binding     *resolution.BindingResult
	projectRoot string
	localCount  map[string]int
	docs        map[string]*scip.Document
	symbolInfos map[string]*scip.SymbolInformation
}

func newDocumentBuilder(graph *resolution.SymbolGraph, binding *resolution.BindingResult, projectRoot string) *documentBuilder {
	return &documentBuilder{
		graph:       graph,
		binding:     binding,
		projectRoot: projectRoot,
		localCount:  make(map[string]int),
		docs:        make(map[string]*scip.Document),
		symbolInfos: make(map[string]*scip.SymbolInformation),
	}
}

func (db *documentBuilder) relativePath(absPath string) string {
	rel, err := filepath.Rel(db.projectRoot, absPath)
	if err != nil {
		return absPath
	}
	return filepath.ToSlash(rel)
}

func (db *documentBuilder) getOrCreateDoc(absPath string) *scip.Document {
	rel := db.relativePath(absPath)
	if doc, ok := db.docs[rel]; ok {
		return doc
	}
	doc := &scip.Document{
		Language:         "apex",
		RelativePath:     rel,
		PositionEncoding: scip.PositionEncoding_UTF8CodeUnitOffsetFromLineStart,
	}
	db.docs[rel] = doc
	return doc
}

func (db *documentBuilder) nextLocal(relPath string) string {
	db.localCount[relPath]++
	return localSymbol(db.localCount[relPath])
}

func (db *documentBuilder) addOccurrence(doc *scip.Document, line, col, endCol int, symbol string, roles int32) {
	doc.Occurrences = append(doc.Occurrences, &scip.Occurrence{
		Range:       []int32{int32(line), int32(col), int32(endCol)},
		Symbol:      symbol,
		SymbolRoles: roles,
	})
}

func (db *documentBuilder) buildDefinitions() {
	for _, cls := range db.graph.AllClasses() {
		if cls.Declaration == nil {
			continue
		}
		if cls.Declaration.FilePath != "" && !isMetadataXML(cls.Declaration.FilePath) {
			doc := db.getOrCreateDoc(cls.Declaration.FilePath)
			sym := scipSymbol(db.graph, cls.ID)
			if sym != "" {
				db.addOccurrence(doc,
					cls.Declaration.Line-1,
					cls.Declaration.Column,
					cls.Declaration.Column+len(cls.Declaration.Name),
					sym,
					int32(scip.SymbolRole_Definition),
				)
				si := db.symbolInfo(sym, cls.Declaration.Name, scipKind(resolution.SymbolKindClass))
				if cls.Declaration.DocComment != "" {
					si.Documentation = []string{cls.Declaration.DocComment}
				}
				db.addImplementsRelationships(si, cls.Implements)
				db.addExtendsRelationship(si, cls.Extends)
			}
		}
		db.registerFields(cls)
		db.registerMethods(cls.Declaration.FilePath, cls.Methods)
		db.registerNestedDeclarations(cls)
	}

	for _, iface := range db.graph.AllInterfaces() {
		if iface.Declaration == nil || iface.Declaration.FilePath == "" {
			continue
		}
		doc := db.getOrCreateDoc(iface.Declaration.FilePath)
		sym := scipSymbol(db.graph, iface.ID)
		if sym == "" {
			continue
		}
		db.addOccurrence(doc,
			iface.Declaration.Line-1,
			iface.Declaration.Column,
			iface.Declaration.Column+len(iface.Declaration.Name),
			sym,
			int32(scip.SymbolRole_Definition),
		)
		si := db.symbolInfo(sym, iface.Declaration.Name, scipKind(resolution.SymbolKindInterface))
		db.addExtendsRelationships(si, iface.Extends)
		db.registerMethods(iface.Declaration.FilePath, iface.Methods)
	}

	for _, enum := range db.graph.AllEnums() {
		if enum.Declaration == nil || enum.Declaration.FilePath == "" {
			continue
		}
		doc := db.getOrCreateDoc(enum.Declaration.FilePath)
		sym := scipSymbol(db.graph, enum.ID)
		if sym == "" {
			continue
		}
		db.addOccurrence(doc,
			enum.Declaration.Line-1,
			enum.Declaration.Column,
			enum.Declaration.Column+len(enum.Declaration.Name),
			sym,
			int32(scip.SymbolRole_Definition),
		)
		db.symbolInfo(sym, enum.Declaration.Name, scipKind(resolution.SymbolKindEnum))
	}

	for _, trigger := range db.graph.AllTriggers() {
		if trigger.Declaration == nil || trigger.Declaration.FilePath == "" {
			continue
		}
		doc := db.getOrCreateDoc(trigger.Declaration.FilePath)
		sym := scipSymbol(db.graph, trigger.ID)
		if sym == "" {
			continue
		}
		db.addOccurrence(doc,
			trigger.Declaration.Line-1,
			trigger.Declaration.Column,
			trigger.Declaration.Column+len(trigger.Declaration.Name),
			sym,
			int32(scip.SymbolRole_Definition),
		)
		db.symbolInfo(sym, trigger.Declaration.Name, scipKind(resolution.SymbolKindTrigger))
	}
}

func (db *documentBuilder) registerFields(cls *resolution.ClassSymbol) {
	if cls.Declaration == nil {
		return
	}
	for _, fieldID := range cls.Fields {
		field, ok := db.graph.FieldByID(fieldID)
		if !ok || field.Declaration == nil {
			continue
		}
		filePath := cls.Declaration.FilePath
		if field.Declaration.FilePath != "" {
			filePath = field.Declaration.FilePath
		}
		if filePath == "" || isMetadataXML(filePath) {
			continue
		}
		doc := db.getOrCreateDoc(filePath)
		sym := scipSymbol(db.graph, fieldID)
		if sym == "" {
			continue
		}
		kind := resolution.SymbolKindField
		if field.Declaration.Getter != nil || field.Declaration.Setter != nil {
			kind = resolution.SymbolKindProperty
		}
		db.addOccurrence(doc,
			field.Declaration.Line-1,
			field.Declaration.Column,
			field.Declaration.Column+len(field.Declaration.Name),
			sym,
			int32(scip.SymbolRole_Definition),
		)
		db.symbolInfo(sym, field.Declaration.Name, scipKind(kind))
	}
}

func (db *documentBuilder) registerMethods(filePath string, methodIDs []resolution.SymbolID) {
	if filePath == "" {
		return
	}
	for _, methodID := range methodIDs {
		method, ok := db.graph.MethodByID(methodID)
		if !ok || method.Declaration == nil {
			continue
		}
		doc := db.getOrCreateDoc(filePath)
		sym := scipSymbol(db.graph, methodID)
		if sym == "" {
			continue
		}
		db.addOccurrence(doc,
			method.Declaration.Line-1,
			method.Declaration.Column,
			method.Declaration.Column+len(method.Declaration.Name),
			sym,
			int32(scip.SymbolRole_Definition),
		)
		si := db.symbolInfo(sym, method.Declaration.Name, scipKind(method.Kind))
		if method.Declaration.DocComment != "" {
			si.Documentation = []string{method.Declaration.DocComment}
		}
		db.registerParameters(filePath, method)
	}
}

func (db *documentBuilder) registerParameters(filePath string, method *resolution.MethodSymbol) {
	if method.Declaration == nil {
		return
	}
	relPath := db.relativePath(filePath)
	for _, paramID := range method.Parameters {
		v, ok := db.graph.VariableByID(paramID)
		if !ok {
			continue
		}
		param, ok := v.Declaration.(*ast.ParameterDeclaration)
		if !ok || param == nil {
			continue
		}
		sym := db.nextLocal(relPath)
		doc := db.getOrCreateDoc(filePath)
		// ParameterDeclaration doesn't have Line/Column, so we skip occurrence
		// but still register the symbol info
		db.symbolInfo(sym, param.Name, scipKind(resolution.SymbolKindParameter))
		_ = doc
	}
}

func (db *documentBuilder) registerNestedDeclarations(cls *resolution.ClassSymbol) {
	// Nested types are already handled by the top-level iteration of AllClasses/AllInterfaces/AllEnums
}

func (db *documentBuilder) buildReferences() {
	if db.binding == nil {
		return
	}
	for expr, symID := range db.binding.Identifiers {
		if expr.Synthetic {
			continue
		}
		filePath := db.binding.NodeFiles[expr]
		if filePath == "" {
			continue
		}
		if db.isStdlib(symID) {
			continue
		}
		sym := db.resolveSymbolString(symID, filePath)
		if sym == "" {
			continue
		}
		doc := db.getOrCreateDoc(filePath)
		db.addOccurrence(doc,
			expr.Line-1,
			expr.Column,
			expr.Column+len(expr.Name),
			sym,
			int32(scip.SymbolRole_ReadAccess),
		)
	}

	for expr, symID := range db.binding.MethodCalls {
		filePath := db.binding.NodeFiles[expr]
		if filePath == "" {
			continue
		}
		if db.isStdlib(symID) {
			continue
		}
		sym := scipSymbol(db.graph, symID)
		if sym == "" {
			continue
		}
		doc := db.getOrCreateDoc(filePath)
		db.addOccurrence(doc,
			expr.Line-1,
			expr.Column,
			expr.Column+len(expr.Name),
			sym,
			int32(scip.SymbolRole_ReadAccess),
		)
	}

	for expr, symID := range db.binding.FieldAccesses {
		filePath := db.binding.NodeFiles[expr]
		if filePath == "" {
			continue
		}
		if db.isStdlib(symID) {
			continue
		}
		sym := scipSymbol(db.graph, symID)
		if sym == "" {
			continue
		}
		doc := db.getOrCreateDoc(filePath)
		db.addOccurrence(doc,
			expr.Line-1,
			expr.Column,
			expr.Column+len(expr.FieldName),
			sym,
			int32(scip.SymbolRole_ReadAccess),
		)
	}

	for expr, cb := range db.binding.ConstructorBindings {
		filePath := db.binding.NodeFiles[expr]
		if filePath == "" {
			continue
		}
		if db.isStdlib(cb.Type) {
			continue
		}
		sym := scipSymbol(db.graph, cb.Type)
		if sym == "" {
			continue
		}
		doc := db.getOrCreateDoc(filePath)
		typeName := expr.TypeName
		if idx := strings.LastIndex(typeName, "."); idx >= 0 {
			typeName = typeName[idx+1:]
		}
		db.addOccurrence(doc,
			expr.Line-1,
			expr.Column,
			expr.Column+len(expr.TypeName),
			sym,
			int32(scip.SymbolRole_ReadAccess),
		)
	}
}

func (db *documentBuilder) resolveSymbolString(symID resolution.SymbolID, filePath string) string {
	sym := scipSymbol(db.graph, symID)
	if sym != "" {
		return sym
	}
	// Variable symbols (parameters, locals) use document-local symbols
	if v, ok := db.graph.VariableByID(symID); ok {
		if v.Kind == resolution.SymbolKindParameter || v.Kind == resolution.SymbolKindLocalVariable {
			relPath := db.relativePath(filePath)
			return db.nextLocal(relPath)
		}
	}
	return ""
}

func (db *documentBuilder) isStdlib(symID resolution.SymbolID) bool {
	if symID == resolution.InvalidSymbol {
		return true
	}
	if cls, ok := db.graph.ClassByID(symID); ok {
		return cls.Declaration == nil || cls.Declaration.FilePath == ""
	}
	if iface, ok := db.graph.InterfaceByID(symID); ok {
		return iface.Declaration == nil || iface.Declaration.FilePath == ""
	}
	if enum, ok := db.graph.EnumByID(symID); ok {
		return enum.Declaration == nil || enum.Declaration.FilePath == ""
	}
	if method, ok := db.graph.MethodByID(symID); ok {
		return db.isStdlib(method.DeclaringType)
	}
	if field, ok := db.graph.FieldByID(symID); ok {
		return db.isStdlib(field.DeclaringType)
	}
	if trigger, ok := db.graph.TriggerByID(symID); ok {
		return trigger.Declaration == nil || trigger.Declaration.FilePath == ""
	}
	if v, ok := db.graph.VariableByID(symID); ok {
		return db.isStdlib(v.Parent)
	}
	return true
}

func (db *documentBuilder) symbolInfo(sym, displayName string, kind scip.SymbolInformation_Kind) *scip.SymbolInformation {
	if si, ok := db.symbolInfos[sym]; ok {
		return si
	}
	si := &scip.SymbolInformation{
		Symbol:      sym,
		DisplayName: displayName,
		Kind:        kind,
	}
	db.symbolInfos[sym] = si
	return si
}

func (db *documentBuilder) addImplementsRelationships(si *scip.SymbolInformation, implements []resolution.SymbolID) {
	for _, ifaceID := range implements {
		if db.isStdlib(ifaceID) {
			continue
		}
		ifaceSym := scipSymbol(db.graph, ifaceID)
		if ifaceSym == "" {
			continue
		}
		si.Relationships = append(si.Relationships, &scip.Relationship{
			Symbol:           ifaceSym,
			IsImplementation: true,
		})
	}
}

func (db *documentBuilder) addExtendsRelationship(si *scip.SymbolInformation, extendsID resolution.SymbolID) {
	if extendsID == resolution.InvalidSymbol || db.isStdlib(extendsID) {
		return
	}
	extSym := scipSymbol(db.graph, extendsID)
	if extSym == "" {
		return
	}
	si.Relationships = append(si.Relationships, &scip.Relationship{
		Symbol:      extSym,
		IsReference: true,
	})
}

func (db *documentBuilder) addExtendsRelationships(si *scip.SymbolInformation, extendsIDs []resolution.SymbolID) {
	for _, extID := range extendsIDs {
		db.addExtendsRelationship(si, extID)
	}
}

func isMetadataXML(filePath string) bool {
	return strings.HasSuffix(filePath, ".object-meta.xml") ||
		strings.HasSuffix(filePath, ".field-meta.xml") ||
		strings.HasSuffix(filePath, ".object")
}

func (db *documentBuilder) documents() []*scip.Document {
	result := make([]*scip.Document, 0, len(db.docs))
	for _, doc := range db.docs {
		// Attach symbol infos to their documents
		for _, si := range db.symbolInfos {
			for _, occ := range doc.Occurrences {
				if occ.Symbol == si.Symbol && occ.SymbolRoles&int32(scip.SymbolRole_Definition) != 0 {
					doc.Symbols = append(doc.Symbols, si)
					break
				}
			}
		}
		result = append(result, doc)
	}
	return result
}
