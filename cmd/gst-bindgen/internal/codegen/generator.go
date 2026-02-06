package codegen

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-gst/go-gst/cmd/gst-bindgen/internal/girparser"
	"github.com/go-gst/go-gst/cmd/gst-bindgen/internal/overrides"
)

// Config holds all configuration for the code generator.
type Config struct {
	GIRDir       string // Directory containing GIR XML files
	OutputDir    string // Root output directory (parent of gst/)
	OverridesDir string // Directory containing override YAML files
}

// PackageConfig maps a GIR namespace to a Go package.
type PackageConfig struct {
	Namespace  string // GIR namespace name (e.g., "Gst")
	Version    string // GIR namespace version (e.g., "1.0")
	GIRFile    string // GIR filename (e.g., "Gst-1.0.gir")
	GoPackage  string // Go package path relative to output dir (e.g., "gst")
	PkgConfig  string // pkg-config package name (e.g., "gstreamer-1.0")
	CPrefix    string // C symbol prefix (e.g., "gst")
	OverrideFile string // Override YAML filename (e.g., "gst.yaml")
}

// DefaultPackages defines the mapping from GIR namespaces to Go packages.
var DefaultPackages = []PackageConfig{
	{Namespace: "Gst", Version: "1.0", GIRFile: "Gst-1.0.gir", GoPackage: "gst", PkgConfig: "gstreamer-1.0", CPrefix: "gst", OverrideFile: "gst.yaml"},
	{Namespace: "GstBase", Version: "1.0", GIRFile: "GstBase-1.0.gir", GoPackage: "gst/base", PkgConfig: "gstreamer-base-1.0", CPrefix: "gst_base", OverrideFile: "gst_base.yaml"},
	{Namespace: "GstApp", Version: "1.0", GIRFile: "GstApp-1.0.gir", GoPackage: "gst/app", PkgConfig: "gstreamer-app-1.0", CPrefix: "gst_app", OverrideFile: "gst_app.yaml"},
	{Namespace: "GstVideo", Version: "1.0", GIRFile: "GstVideo-1.0.gir", GoPackage: "gst/video", PkgConfig: "gstreamer-video-1.0", CPrefix: "gst_video", OverrideFile: "gst_video.yaml"},
	{Namespace: "GstAudio", Version: "1.0", GIRFile: "GstAudio-1.0.gir", GoPackage: "gst/audio", PkgConfig: "gstreamer-audio-1.0", CPrefix: "gst_audio", OverrideFile: "gst_audio.yaml"},
	{Namespace: "GstPbutils", Version: "1.0", GIRFile: "GstPbutils-1.0.gir", GoPackage: "gst/pbutils", PkgConfig: "gstreamer-pbutils-1.0", CPrefix: "gst_pbutils", OverrideFile: "gst_pbutils.yaml"},
	{Namespace: "GstRtp", Version: "1.0", GIRFile: "GstRtp-1.0.gir", GoPackage: "gst/rtp", PkgConfig: "gstreamer-rtp-1.0", CPrefix: "gst_rtp", OverrideFile: "gst_rtp.yaml"},
	{Namespace: "GstSdp", Version: "1.0", GIRFile: "GstSdp-1.0.gir", GoPackage: "gst/gstsdp", PkgConfig: "gstreamer-sdp-1.0", CPrefix: "gst_sdp", OverrideFile: "gst_sdp.yaml"},
	{Namespace: "GstWebRTC", Version: "1.0", GIRFile: "GstWebRTC-1.0.gir", GoPackage: "gst/gstwebrtc", PkgConfig: "gstreamer-webrtc-1.0", CPrefix: "gst_webrtc", OverrideFile: "gst_webrtc.yaml"},
	{Namespace: "GstNet", Version: "1.0", GIRFile: "GstNet-1.0.gir", GoPackage: "gst/gstnet", PkgConfig: "gstreamer-net-1.0", CPrefix: "gst_net", OverrideFile: "gst_net.yaml"},
}

// Generate reads GIR files, applies overrides, and generates Go binding source files.
func Generate(cfg *Config) error {
	// Parse all GIR files and build a type registry.
	registry, err := parseAllGIR(cfg.GIRDir)
	if err != nil {
		return fmt.Errorf("parsing GIR files: %w", err)
	}

	for _, pkg := range DefaultPackages {
		girPath := filepath.Join(cfg.GIRDir, pkg.GIRFile)
		if _, err := os.Stat(girPath); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "skipping %s: GIR file not found\n", pkg.GIRFile)
			continue
		}

		// Load overrides for this package.
		ovr, err := loadOverrides(cfg.OverridesDir, pkg.OverrideFile)
		if err != nil {
			return fmt.Errorf("loading overrides for %s: %w", pkg.GoPackage, err)
		}

		// Generate code for this package.
		if err := generatePackage(cfg, pkg, registry, ovr); err != nil {
			return fmt.Errorf("generating %s: %w", pkg.GoPackage, err)
		}
	}

	return nil
}

// Validate regenerates code and checks that it matches the committed files.
func Validate(cfg *Config) error {
	// TODO: Implement validation by generating to a temp dir and diffing.
	return fmt.Errorf("validate not yet implemented")
}

func parseAllGIR(girDir string) (*girparser.TypeRegistry, error) {
	registry := girparser.NewTypeRegistry()

	entries, err := os.ReadDir(girDir)
	if err != nil {
		return nil, fmt.Errorf("reading GIR directory %s: %w", girDir, err)
	}

	for _, entry := range entries {
		if filepath.Ext(entry.Name()) != ".gir" {
			continue
		}
		path := filepath.Join(girDir, entry.Name())
		repo, err := girparser.ParseRepository(path)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", entry.Name(), err)
		}
		registry.AddRepository(repo)
	}

	if err := registry.Resolve(); err != nil {
		return nil, fmt.Errorf("resolving types: %w", err)
	}

	return registry, nil
}

func loadOverrides(overridesDir, filename string) (*overrides.PackageOverrides, error) {
	path := filepath.Join(overridesDir, filename)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// No overrides file — return empty overrides.
		return &overrides.PackageOverrides{}, nil
	}
	return overrides.LoadFile(path)
}

func generatePackage(cfg *Config, pkg PackageConfig, registry *girparser.TypeRegistry, ovr *overrides.PackageOverrides) error {
	ns := registry.Namespace(pkg.Namespace, pkg.Version)
	if ns == nil {
		return fmt.Errorf("namespace %s-%s not found in type registry", pkg.Namespace, pkg.Version)
	}

	outDir := filepath.Join(cfg.OutputDir, pkg.GoPackage)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("creating output directory %s: %w", outDir, err)
	}

	ctx := &PackageContext{
		Config:    cfg,
		Package:   pkg,
		Registry:  registry,
		Namespace: ns,
		Overrides: ovr,
		OutputDir: outDir,
	}

	// Phase 1: Generate enums and constants.
	if err := generateEnums(ctx); err != nil {
		return fmt.Errorf("generating enums: %w", err)
	}

	// Future phases will add more generation steps here.

	return nil
}

// PackageContext holds all state needed to generate code for a single package.
type PackageContext struct {
	Config    *Config
	Package   PackageConfig
	Registry  *girparser.TypeRegistry
	Namespace *girparser.Namespace
	Overrides *overrides.PackageOverrides
	OutputDir string
}
