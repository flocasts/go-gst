package codegen

import (
	"strings"
	"unicode"
)

// knownAcronyms are words that should stay fully uppercase in Go names.
var knownAcronyms = map[string]bool{
	"URI":  true,
	"URL":  true,
	"TOC":  true,
	"RTP":  true,
	"RTSP": true,
	"SDP":  true,
	"GL":   true,
	"EOS":  true,
	"ID":   true,
	"IO":   true,
	"IP":   true,
	"HTTP": true,
	"RTCP": true,
	"DTLS": true,
	"SRTP": true,
	"SCTP": true,
	"ICE":  true,
	"SRC":  true,
}

// CTypeToGoType converts a C type name to a Go type name by stripping the
// namespace prefix. For example, "GstElement" becomes "Element".
func CTypeToGoType(cType, nsPrefix string) string {
	name := cType
	// Strip pointer suffix.
	name = strings.TrimRight(name, " *")
	// Strip the namespace prefix (e.g., "Gst" from "GstElement").
	if nsPrefix != "" && strings.HasPrefix(name, nsPrefix) {
		name = name[len(nsPrefix):]
	}
	return name
}

// GIRNameToGoType converts a GIR type name to a Go type name.
// GIR names are already without namespace prefix (e.g., "Element").
func GIRNameToGoType(girName string) string {
	// Most GIR names are already PascalCase.
	return girName
}

// CSymbolToMethodName converts a C function symbol to a Go method name by
// stripping the prefix that matches the type and converting to PascalCase.
// For example, "gst_element_get_name" with prefix "gst_element_" becomes "GetName".
func CSymbolToMethodName(cSymbol, prefix string) string {
	name := cSymbol
	if prefix != "" && strings.HasPrefix(name, prefix) {
		name = name[len(prefix):]
	}
	return UnderscoreToPascal(name)
}

// CSymbolToFuncName converts a C function symbol to a Go package-level function name.
// For example, "gst_element_factory_make" with prefix "gst_" becomes "ElementFactoryMake".
func CSymbolToFuncName(cSymbol, nsPrefix string) string {
	name := cSymbol
	if nsPrefix != "" && strings.HasPrefix(name, nsPrefix) {
		name = name[len(nsPrefix):]
	}
	return UnderscoreToPascal(name)
}

// CEnumMemberToGoConst converts a C enum member identifier to a Go constant name.
// For example, "GST_STATE_PLAYING" becomes "StatePlaying".
func CEnumMemberToGoConst(cIdentifier, nsUpper string) string {
	name := cIdentifier
	// Strip the namespace prefix (e.g., "GST_" from "GST_STATE_PLAYING").
	prefix := nsUpper + "_"
	if strings.HasPrefix(name, prefix) {
		name = name[len(prefix):]
	}
	return UnderscoreToPascal(name)
}

// UnderscoreToPascal converts an underscore_separated name to PascalCase,
// handling known acronyms.
func UnderscoreToPascal(name string) string {
	parts := strings.Split(name, "_")
	var result strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		upper := strings.ToUpper(part)
		if knownAcronyms[upper] {
			result.WriteString(upper)
		} else {
			result.WriteString(capitalizeFirst(part))
		}
	}
	return result.String()
}

// capitalizeFirst returns the string with the first letter capitalized and
// the rest lowercased.
func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(strings.ToLower(s))
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

// GoTypeToReceiverName returns a short receiver name for a Go type.
// For example, "Element" becomes "e", "Pipeline" becomes "p".
func GoTypeToReceiverName(goType string) string {
	if goType == "" {
		return "v"
	}
	return strings.ToLower(goType[:1])
}

// ConstructorName returns the Go function name for a constructor.
// For example, type "Pipeline" and GIR constructor "new" becomes "NewPipeline".
func ConstructorName(goType, girConstructorName string) string {
	if girConstructorName == "new" {
		return "New" + goType
	}
	// For other constructors like "new_from_uri", produce "NewPipelineFromURI".
	suffix := girConstructorName
	if strings.HasPrefix(suffix, "new_") {
		suffix = suffix[4:]
	}
	return "New" + goType + UnderscoreToPascal(suffix)
}

// NSPrefixUpper returns the uppercase prefix used in C constants.
// For example, namespace "Gst" returns "GST", namespace "GstBase" returns "GST_BASE".
func NSPrefixUpper(nsName string) string {
	// Convert from CamelCase to UPPER_SNAKE, grouping consecutive uppercase letters.
	runes := []rune(nsName)
	var result strings.Builder
	for i, r := range runes {
		if i > 0 && unicode.IsUpper(r) {
			prev := runes[i-1]
			if unicode.IsLower(prev) {
				result.WriteByte('_')
			}
			// Don't insert underscore between consecutive uppercase letters.
		}
		result.WriteRune(unicode.ToUpper(r))
	}
	return result.String()
}

// NSPrefixLower returns the lowercase prefix used in C function symbols.
// For example, namespace "Gst" returns "gst_", namespace "GstBase" returns "gst_base_".
func NSPrefixLower(nsName string) string {
	return strings.ToLower(NSPrefixUpper(nsName)) + "_"
}

// CTypeToCastMacro converts a C type to the expected GLib type-check cast macro.
// For example, "GstElement" becomes "GST_ELEMENT".
func CTypeToCastMacro(cType string) string {
	// Convert from CamelCase to UPPER_SNAKE, but handle the common patterns.
	var result strings.Builder
	for i, r := range cType {
		if i > 0 && unicode.IsUpper(r) {
			// Don't insert underscore between consecutive uppercase letters
			// unless the next char is lowercase (e.g., "GstURI" -> "GST_URI" not "GST_U_R_I").
			prev := rune(cType[i-1])
			if unicode.IsLower(prev) {
				result.WriteByte('_')
			} else if i+1 < len(cType) && unicode.IsLower(rune(cType[i+1])) {
				result.WriteByte('_')
			}
		}
		result.WriteRune(unicode.ToUpper(r))
	}
	return result.String()
}
