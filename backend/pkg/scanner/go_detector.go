package scanner

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"log"
	"path/filepath"
	"strings"
)

// AnalyzeGoRepo performs an AST-based scan of Go files to find
// Prometheus and OpenTelemetry usage and returns file-level candidates.
func AnalyzeGoRepo(root string) ([]Candidate, error) {
	log.Printf("[scanner][go] AnalyzeGoRepo started for %s", root)
	var metricFilesSet = map[string]struct{}{}
	var otelFilesSet = map[string]struct{}{}

	walkFn := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// skip vendor and .git
			base := filepath.Base(path)
			if base == "vendor" || base == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		fset := token.NewFileSet()
		fileAst, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			// skip files that fail to parse
			return nil
		}

		// Collect imports
		imports := map[string]struct{}{}
		importAliases := map[string]string{} // alias -> import path
		for _, imp := range fileAst.Imports {
			ip := strings.Trim(imp.Path.Value, `"`)
			imports[ip] = struct{}{}
			if imp.Name != nil {
				importAliases[imp.Name.Name] = ip
			} else {
				// infer default alias (last element)
				parts := strings.Split(ip, "/")
				alias := parts[len(parts)-1]
				importAliases[alias] = ip
			}
		}

		hasPromImport := false
		hasPromhttpImport := false
		hasOtelImport := false
		hasOtelGinImport := false

		for ip := range imports {
			if strings.Contains(ip, "prometheus/client_golang") {
				hasPromImport = true
			}
			if strings.Contains(ip, "prometheus/client_golang/prometheus/promhttp") || strings.Contains(ip, "prometheus/client_golang/prometheus") {
				hasPromhttpImport = true
			}
			if strings.Contains(ip, "opentelemetry") {
				hasOtelImport = true
			}
			if strings.Contains(ip, "otelgin") || strings.Contains(ip, "contrib/instrumentation/github.com/gin-gonic/gin/otelgin") {
				hasOtelGinImport = true
			}
		}

		var foundPromUsage bool
		var foundPromHandler bool
		var foundPromRegister bool
		var foundGinDefault bool
		var foundOtelInit bool
		var foundOtelMiddleware bool
		var foundOtelSetTracerProvider bool

		ast.Inspect(fileAst, func(n ast.Node) bool {
			switch expr := n.(type) {
			case *ast.CallExpr:
				// Identify selector calls like promhttp.Handler() or prometheus.MustRegister()
				if sel, ok := expr.Fun.(*ast.SelectorExpr); ok {
					if ident, ok := sel.X.(*ast.Ident); ok {
						// e.g., promhttp.Handler()
						if sel.Sel != nil {
							name := sel.Sel.Name
							xname := ident.Name
							// promhttp.Handler
							if name == "Handler" && (strings.Contains(xname, "promhttp") || hasPromhttpImport) {
								foundPromHandler = true
							}
							// prometheus.MustRegister
							if name == "MustRegister" && (strings.Contains(xname, "prometheus") || hasPromImport) {
								foundPromRegister = true
							}
							if (name == "NewCounterVec" || name == "NewHistogramVec" || name == "Counter" || name == "Histogram") && (strings.Contains(xname, "prometheus") || hasPromImport) {
								foundPromUsage = true
							}
							// gin.Default() detection
							if name == "Default" && (xname == "gin" || strings.Contains(xname, "gin")) {
								foundGinDefault = true
							}
							// otelgin.Middleware detection (e.g., otelgin.Middleware("svc"))
							if name == "Middleware" && (strings.Contains(xname, "otelgin") || hasOtelGinImport) {
								foundOtelMiddleware = true
							}
							// otel.SetTracerProvider detection
							if name == "SetTracerProvider" && (strings.Contains(xname, "otel") || hasOtelImport) {
								foundOtelSetTracerProvider = true
							}
							// otel tracer provider or otlptrace usage
							if (name == "NewTracerProvider" || strings.Contains(name, "otlptrace") || name == "Start") && hasOtelImport {
								foundOtelInit = true
							}
						}
					}
					// If nested selector like sdktrace.NewTracerProvider
					if xsel, ok := sel.X.(*ast.SelectorExpr); ok {
						if xsel.Sel != nil && sel.Sel != nil {
							if sel.Sel.Name == "NewTracerProvider" && strings.Contains(xsel.Sel.Name, "sdktrace") {
								foundOtelInit = true
							}
						}
					}
				}
			}
			return true
		})

		// Mark metrics files only when actual Prometheus code is used (not import-only)
		if foundPromHandler || foundPromRegister || foundPromUsage {
			metricFilesSet[path] = struct{}{}
		}

		// Mark OTel candidate files when tracer is initialized or otel middleware is used.
		// Only treat `gin.Default()` as an OTel signal if there are explicit otel imports
		// (otherwise a plain Gin server shouldn't imply tracing is already present).
		if foundOtelInit || foundOtelMiddleware || foundOtelSetTracerProvider {
			otelFilesSet[path] = struct{}{}
		} else if foundGinDefault && (hasOtelImport || hasOtelGinImport) {
			// only consider gin.Default as OTel candidate when otel imports exist
			otelFilesSet[path] = struct{}{}
		}

		_ = importAliases // kept for more advanced analysis later
		return nil
	}

	if err := filepath.WalkDir(root, walkFn); err != nil {
		return nil, fmt.Errorf("walk failed: %w", err)
	}

	var metricFiles []string
	var otelFiles []string
	for f := range metricFilesSet {
		metricFiles = append(metricFiles, strings.TrimPrefix(f, root+"/"))
	}
	for f := range otelFilesSet {
		otelFiles = append(otelFiles, strings.TrimPrefix(f, root+"/"))
	}

	var candidates []Candidate
	if len(metricFiles) > 0 {
		candidates = append(candidates, Candidate{
			Language:    "Go",
			Framework:   "Go",
			Kind:        "metrics",
			Patterns:    []string{"prometheus.MustRegister", "promhttp.Handler", "NewCounterVec"},
			Files:       metricFiles,
			ServiceName: "go-service",
		})
	}
	if len(otelFiles) > 0 {
		candidates = append(candidates, Candidate{
			Language:    "Go",
			Framework:   "Go",
			Kind:        "otel",
			Patterns:    []string{"gin.Default", "sdktrace.NewTracerProvider", "otlptrace"},
			Files:       otelFiles,
			ServiceName: "go-service",
		})
	}

	log.Printf("[scanner][go] AnalyzeGoRepo complete for %s: metrics=%d otel=%d", root, len(metricFiles), len(otelFiles))
	return candidates, nil
}
