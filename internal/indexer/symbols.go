package indexer

import (
	"fmt"
	"strings"

	"github.com/octoberswimmer/aer/resolution"
)

const (
	scipScheme  = "scip-apex"
	scipManager = "apex"
	scipPackage = "."
	scipVersion = "."
)

func symbolPrefix() string {
	return scipScheme + " " + scipManager + " " + scipPackage + " " + scipVersion + " "
}

func scipSymbol(graph *resolution.SymbolGraph, id resolution.SymbolID) string {
	if id == resolution.InvalidSymbol {
		return ""
	}
	descriptors := descriptorsFor(graph, id)
	if descriptors == "" {
		return ""
	}
	return symbolPrefix() + descriptors
}

func descriptorsFor(graph *resolution.SymbolGraph, id resolution.SymbolID) string {
	if cls, ok := graph.ClassByID(id); ok {
		return typeDescriptors(cls.QualifiedName)
	}
	if iface, ok := graph.InterfaceByID(id); ok {
		return typeDescriptors(iface.QualifiedName)
	}
	if enum, ok := graph.EnumByID(id); ok {
		return typeDescriptors(enum.QualifiedName)
	}
	if method, ok := graph.MethodByID(id); ok {
		parent := descriptorsFor(graph, method.DeclaringType)
		if parent == "" {
			return ""
		}
		name := escapeName(method.Name)
		if method.Kind == resolution.SymbolKindConstructor {
			name = "`<init>`"
		}
		return parent + name + "()."
	}
	if field, ok := graph.FieldByID(id); ok {
		parent := descriptorsFor(graph, field.DeclaringType)
		if parent == "" {
			return ""
		}
		return parent + escapeName(field.Name) + "."
	}
	if trigger, ok := graph.TriggerByID(id); ok {
		return typeDescriptors(trigger.QualifiedName)
	}
	return ""
}

func typeDescriptors(qualifiedName string) string {
	parts := strings.Split(qualifiedName, ".")
	var b strings.Builder
	for _, part := range parts {
		b.WriteString(escapeName(part))
		b.WriteByte('#')
	}
	return b.String()
}

func localSymbol(n int) string {
	return fmt.Sprintf("local %d", n)
}

func escapeName(name string) string {
	for _, ch := range name {
		if !isIdentChar(ch) {
			return "`" + strings.ReplaceAll(name, "`", "``") + "`"
		}
	}
	return name
}

func isIdentChar(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') ||
		(ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9') ||
		ch == '_' || ch == '+' || ch == '-' || ch == '$'
}
