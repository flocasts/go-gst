package codegen

import "testing"

func TestCTypeToGoType(t *testing.T) {
	tests := []struct {
		cType    string
		nsPrefix string
		want     string
	}{
		{"GstElement", "Gst", "Element"},
		{"GstPipeline", "Gst", "Pipeline"},
		{"GstBaseSrc", "GstBase", "Src"},
		{"GstAppSink", "GstApp", "Sink"},
		{"GstElement *", "Gst", "Element"},
	}
	for _, tt := range tests {
		got := CTypeToGoType(tt.cType, tt.nsPrefix)
		if got != tt.want {
			t.Errorf("CTypeToGoType(%q, %q) = %q, want %q", tt.cType, tt.nsPrefix, got, tt.want)
		}
	}
}

func TestCEnumMemberToGoConst(t *testing.T) {
	tests := []struct {
		cIdentifier string
		nsUpper     string
		want        string
	}{
		{"GST_STATE_PLAYING", "GST", "StatePlaying"},
		{"GST_STATE_VOID_PENDING", "GST", "StateVoidPending"},
		{"GST_EVENT_FLUSH_START", "GST", "EventFlushStart"},
		{"GST_PAD_SRC", "GST", "PadSRC"},
		{"GST_FORMAT_UNDEFINED", "GST", "FormatUndefined"},
		{"GST_URI_UNKNOWN", "GST", "URIUnknown"},
	}
	for _, tt := range tests {
		got := CEnumMemberToGoConst(tt.cIdentifier, tt.nsUpper)
		if got != tt.want {
			t.Errorf("CEnumMemberToGoConst(%q, %q) = %q, want %q", tt.cIdentifier, tt.nsUpper, got, tt.want)
		}
	}
}

func TestUnderscoreToPascal(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"get_name", "GetName"},
		{"set_state", "SetState"},
		{"void_pending", "VoidPending"},
		{"flush_start", "FlushStart"},
		{"src", "SRC"},
		{"uri", "URI"},
		{"eos", "EOS"},
		{"rtp", "RTP"},
		{"new_from_uri", "NewFromURI"},
	}
	for _, tt := range tests {
		got := UnderscoreToPascal(tt.input)
		if got != tt.want {
			t.Errorf("UnderscoreToPascal(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestCSymbolToMethodName(t *testing.T) {
	tests := []struct {
		cSymbol string
		prefix  string
		want    string
	}{
		{"gst_element_get_name", "gst_element_", "GetName"},
		{"gst_element_set_state", "gst_element_", "SetState"},
		{"gst_object_get_parent", "gst_object_", "GetParent"},
	}
	for _, tt := range tests {
		got := CSymbolToMethodName(tt.cSymbol, tt.prefix)
		if got != tt.want {
			t.Errorf("CSymbolToMethodName(%q, %q) = %q, want %q", tt.cSymbol, tt.prefix, got, tt.want)
		}
	}
}

func TestConstructorName(t *testing.T) {
	tests := []struct {
		goType string
		girName string
		want   string
	}{
		{"Pipeline", "new", "NewPipeline"},
		{"Pipeline", "new_from_uri", "NewPipelineFromURI"},
		{"Element", "new", "NewElement"},
	}
	for _, tt := range tests {
		got := ConstructorName(tt.goType, tt.girName)
		if got != tt.want {
			t.Errorf("ConstructorName(%q, %q) = %q, want %q", tt.goType, tt.girName, got, tt.want)
		}
	}
}

func TestNSPrefixUpper(t *testing.T) {
	tests := []struct {
		nsName string
		want   string
	}{
		{"Gst", "GST"},
		{"GstBase", "GST_BASE"},
		{"GstApp", "GST_APP"},
		{"GstVideo", "GST_VIDEO"},
		{"GstWebRTC", "GST_WEB_RTC"},
	}
	for _, tt := range tests {
		got := NSPrefixUpper(tt.nsName)
		if got != tt.want {
			t.Errorf("NSPrefixUpper(%q) = %q, want %q", tt.nsName, got, tt.want)
		}
	}
}

func TestCTypeToCastMacro(t *testing.T) {
	tests := []struct {
		cType string
		want  string
	}{
		{"GstElement", "GST_ELEMENT"},
		{"GstPipeline", "GST_PIPELINE"},
		{"GstBin", "GST_BIN"},
		{"GstObject", "GST_OBJECT"},
		{"GstBufferPool", "GST_BUFFER_POOL"},
		{"GstURIHandler", "GST_URI_HANDLER"},
	}
	for _, tt := range tests {
		got := CTypeToCastMacro(tt.cType)
		if got != tt.want {
			t.Errorf("CTypeToCastMacro(%q) = %q, want %q", tt.cType, got, tt.want)
		}
	}
}
