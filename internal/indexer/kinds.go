package indexer

import (
	"github.com/octoberswimmer/aer/resolution"
	scip "github.com/sourcegraph/scip/bindings/go/scip"
)

func scipKind(kind resolution.SymbolKind) scip.SymbolInformation_Kind {
	switch kind {
	case resolution.SymbolKindClass:
		return scip.SymbolInformation_Class
	case resolution.SymbolKindInterface:
		return scip.SymbolInformation_Interface
	case resolution.SymbolKindEnum:
		return scip.SymbolInformation_Enum
	case resolution.SymbolKindMethod:
		return scip.SymbolInformation_Method
	case resolution.SymbolKindConstructor:
		return scip.SymbolInformation_Constructor
	case resolution.SymbolKindField:
		return scip.SymbolInformation_Field
	case resolution.SymbolKindProperty:
		return scip.SymbolInformation_Property
	case resolution.SymbolKindParameter:
		return scip.SymbolInformation_Parameter
	case resolution.SymbolKindLocalVariable:
		return scip.SymbolInformation_Variable
	case resolution.SymbolKindTrigger:
		return scip.SymbolInformation_Class
	case resolution.SymbolKindEnumValue:
		return scip.SymbolInformation_EnumMember
	default:
		return scip.SymbolInformation_UnspecifiedKind
	}
}
