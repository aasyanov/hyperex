// HyperEx Playground — interactive web REPL for exploring HyperEx.
package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aasyanov/hyperex"
)

//go:embed web/*
var webFS embed.FS

const (
	Version   = "2.0.0"
	BuildDate = "2026-08-10"
)

var config Config

func main() {
	port := flag.Int("port", 9090, "HTTP server port")
	verbose := flag.Bool("verbose", false, "Enable verbose logging")
	flag.Parse()

	config = Config{
		Port:    *port,
		Verbose: *verbose,
	}

	printBanner()

	mux := http.NewServeMux()

	mux.HandleFunc("/api/eval", handleEval)
	mux.HandleFunc("/api/analyze", handleAnalyze)
	mux.HandleFunc("/api/profile", handleProfile)
	mux.HandleFunc("/api/simplify", handleSimplify)
	mux.HandleFunc("/api/expand", handleExpand)
	mux.HandleFunc("/api/inline", handleInline)
	mux.HandleFunc("/api/validate", handleValidate)
	mux.HandleFunc("/api/extractVariables", handleExtractVariables)
	mux.HandleFunc("/api/extractFunctions", handleExtractFunctions)
	mux.HandleFunc("/api/typecheck", handleTypecheck)
	mux.HandleFunc("/api/examples", handleExamples)
	mux.HandleFunc("/api/functions", handleFunctions)

	webContent, err := fs.Sub(webFS, "web")
	if err != nil {
		log.Fatal(err)
	}
	mux.Handle("/", http.FileServer(http.FS(webContent)))

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", config.Port),
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(sigChan)
		<-sigChan
		log.Println("Shutting down server...")
		_ = server.Close()
	}()

	log.Printf("HyperEx Playground listening at http://localhost:%d", config.Port)
	log.Printf("Press Ctrl+C to stop")

	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}

func printBanner() {
	fmt.Printf(`
===========================================================================
  HyperEx Playground v%s (%s)
  Interactive Web REPL for the HyperEx expression engine

  Features:
    - 70+ built-in functions
    - Sandbox execution with limits
    - Static analysis, profiling, transforms
    - 200+ ready examples
===========================================================================
`, Version, BuildDate)
}

// --- API request/response types ---

type EvalRequest struct {
	Expression string         `json:"expression"`
	Env        map[string]any `json:"env,omitempty"`
	Options    EvalOptions    `json:"options,omitempty"`
}

type EvalOptions struct {
	Sandbox      bool  `json:"sandbox"`
	StrictMode   *bool `json:"strictMode,omitempty"`
	Optimize     *bool `json:"optimize,omitempty"`
	MaxOps       int   `json:"maxOps,omitempty"`
	MaxCallDepth int   `json:"maxCallDepth,omitempty"`
	TimeoutMs    int   `json:"timeoutMs,omitempty"`
}

type EvalResponse struct {
	Success     bool           `json:"success"`
	Result      any            `json:"result,omitempty"`
	ResultType  string         `json:"resultType,omitempty"`
	Error       string         `json:"error,omitempty"`
	CompileTime string         `json:"compileTime,omitempty"`
	ExecTime    string         `json:"execTime,omitempty"`
	TotalTime   string         `json:"totalTime,omitempty"`
	Stats       *hyperex.Stats `json:"stats,omitempty"`
}

type AnalyzeRequest struct {
	Expression string `json:"expression"`
}

type AnalyzeResponse struct {
	Success              bool     `json:"success"`
	Error                string   `json:"error,omitempty"`
	Variables            []string `json:"variables,omitempty"`
	Functions            []string `json:"functions,omitempty"`
	Operators            []string `json:"operators,omitempty"`
	Complexity           string   `json:"complexity,omitempty"`
	IsPure               bool     `json:"isPure,omitempty"`
	NodeCount            int      `json:"nodeCount,omitempty"`
	Depth                int      `json:"depth,omitempty"`
	HasAssignment        bool     `json:"hasAssignment,omitempty"`
	UsesOptionalChaining bool     `json:"usesOptionalChaining,omitempty"`
	UsesNilCoalescing    bool     `json:"usesNilCoalescing,omitempty"`
	HasSideEffects       bool     `json:"hasSideEffects,omitempty"`
	EstimatedCost        int      `json:"estimatedCost,omitempty"`
}

type ProfileRequest struct {
	Expression string         `json:"expression"`
	Env        map[string]any `json:"env,omitempty"`
}

type ProfileResponse struct {
	Success       bool   `json:"success"`
	Error         string `json:"error,omitempty"`
	Result        any    `json:"result,omitempty"`
	ResultType    string `json:"resultType,omitempty"`
	CompileTime   string `json:"compileTime,omitempty"`
	ExecTime      string `json:"execTime,omitempty"`
	OpsExecuted   int64  `json:"opsExecuted,omitempty"`
	MaxStackDepth int    `json:"maxStackDepth,omitempty"`
}

type TransformRequest struct {
	Expression string            `json:"expression"`
	Macros     map[string]string `json:"macros,omitempty"`
}

type TransformResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
	Result  string `json:"result,omitempty"`
}

type ValidateRequest struct {
	Expression string `json:"expression"`
}

type ValidateResponse struct {
	Success bool   `json:"success"`
	Valid   bool   `json:"valid"`
	Error   string `json:"error,omitempty"`
}

type ExtractRequest struct {
	Expression string `json:"expression"`
}

type ExtractResponse struct {
	Success bool     `json:"success"`
	Error   string   `json:"error,omitempty"`
	Names   []string `json:"names,omitempty"`
}

type TypecheckRequest struct {
	Expression string         `json:"expression"`
	Env        map[string]any `json:"env,omitempty"`
}

type TypecheckResponse struct {
	Success   bool     `json:"success"`
	Valid     bool     `json:"valid"`
	Error     string   `json:"error,omitempty"`
	Variables []string `json:"variables,omitempty"`
	Functions []string `json:"functions,omitempty"`
	Unbound   []string `json:"unbound,omitempty"`
}

// --- Handlers ---

func handleEval(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, litMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	var req EvalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, EvalResponse{Success: false, Error: litInvalidRequest + err.Error()})
		return
	}
	if req.Expression == "" {
		writeJSON(w, EvalResponse{Success: false, Error: "Expression is required"})
		return
	}

	startTotal := time.Now()
	engine := buildEngine(req.Options)

	startCompile := time.Now()
	program, err := engine.Compile(req.Expression)
	compileTime := time.Since(startCompile)
	if err != nil {
		stats := engine.Stats()
		writeJSON(w, EvalResponse{
			Success:     false,
			Error:       err.Error(),
			CompileTime: compileTime.String(),
			TotalTime:   time.Since(startTotal).String(),
			Stats:       &stats,
		})
		return
	}

	startExec := time.Now()
	result, err := program.Eval(r.Context(), req.Env)
	execTime := time.Since(startExec)
	stats := engine.Stats()
	if err != nil {
		writeJSON(w, EvalResponse{
			Success:     false,
			Error:       err.Error(),
			CompileTime: compileTime.String(),
			ExecTime:    execTime.String(),
			TotalTime:   time.Since(startTotal).String(),
			Stats:       &stats,
		})
		return
	}

	writeJSON(w, EvalResponse{
		Success:     true,
		Result:      result,
		ResultType:  getTypeName(result),
		CompileTime: compileTime.String(),
		ExecTime:    execTime.String(),
		TotalTime:   time.Since(startTotal).String(),
		Stats:       &stats,
	})
}

func buildEngine(opts EvalOptions) *hyperex.Engine {
	var engineOpts []hyperex.Option

	if opts.Optimize != nil {
		engineOpts = append(engineOpts, hyperex.WithOptimize(*opts.Optimize))
	}
	if opts.StrictMode != nil {
		engineOpts = append(engineOpts, hyperex.WithStrictMode(*opts.StrictMode))
	}
	if opts.MaxCallDepth > 0 {
		engineOpts = append(engineOpts, hyperex.WithMaxCallDepth(opts.MaxCallDepth))
	}
	if opts.Sandbox {
		sandboxCfg := hyperex.SandboxConfig{}
		if opts.MaxOps > 0 {
			sandboxCfg.MaxOps = int64(opts.MaxOps)
		} else {
			sandboxCfg.MaxOps = 100000
		}
		if opts.TimeoutMs > 0 {
			sandboxCfg.Timeout = time.Duration(opts.TimeoutMs) * time.Millisecond
		} else {
			sandboxCfg.Timeout = 5 * time.Second
		}
		engineOpts = append(engineOpts, hyperex.WithSandbox(sandboxCfg))
	}

	return hyperex.New(engineOpts...)
}

func handleAnalyze(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, litMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	var req AnalyzeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, AnalyzeResponse{Success: false, Error: litInvalidRequest + err.Error()})
		return
	}

	analysis, err := hyperex.Analyze(req.Expression)
	if err != nil {
		writeJSON(w, AnalyzeResponse{Success: false, Error: err.Error()})
		return
	}

	writeJSON(w, AnalyzeResponse{
		Success:              true,
		Variables:            analysis.Variables,
		Functions:            analysis.Functions,
		Operators:            analysis.Operators,
		Complexity:           analysis.Complexity.String(),
		IsPure:               analysis.IsPure,
		NodeCount:            analysis.NodeCount,
		Depth:                analysis.Depth,
		HasAssignment:        analysis.HasAssignment,
		UsesOptionalChaining: analysis.UsesOptionalChaining,
		UsesNilCoalescing:    analysis.UsesNilCoalescing,
		HasSideEffects:       analysis.HasSideEffects,
		EstimatedCost:        analysis.EstimatedCost,
	})
}

func handleProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, litMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	var req ProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, ProfileResponse{Success: false, Error: litInvalidRequest + err.Error()})
		return
	}

	profile, result, err := hyperex.ProfileExpr(r.Context(), req.Expression, req.Env)
	if err != nil {
		writeJSON(w, ProfileResponse{Success: false, Error: err.Error()})
		return
	}

	writeJSON(w, ProfileResponse{
		Success:       true,
		Result:        result,
		ResultType:    getTypeName(result),
		CompileTime:   profile.CompileTime.String(),
		ExecTime:      profile.ExecuteTime.String(),
		OpsExecuted:   profile.OpsCount,
		MaxStackDepth: profile.MaxStackDepth,
	})
}

func handleSimplify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, litMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	var req TransformRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, TransformResponse{Success: false, Error: litInvalidRequest + err.Error()})
		return
	}

	result, err := hyperex.Simplify(req.Expression)
	if err != nil {
		writeJSON(w, TransformResponse{Success: false, Error: err.Error()})
		return
	}
	writeJSON(w, TransformResponse{Success: true, Result: result})
}

func handleExpand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, litMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	var req TransformRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, TransformResponse{Success: false, Error: litInvalidRequest + err.Error()})
		return
	}
	if len(req.Macros) == 0 {
		writeJSON(w, TransformResponse{Success: false, Error: "Macros are required for expand"})
		return
	}

	result, err := hyperex.Expand(req.Expression, req.Macros)
	if err != nil {
		writeJSON(w, TransformResponse{Success: false, Error: err.Error()})
		return
	}
	writeJSON(w, TransformResponse{Success: true, Result: result})
}

func handleInline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, litMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	var req TransformRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, TransformResponse{Success: false, Error: litInvalidRequest + err.Error()})
		return
	}

	result, err := hyperex.Inline(req.Expression)
	if err != nil {
		writeJSON(w, TransformResponse{Success: false, Error: err.Error()})
		return
	}
	writeJSON(w, TransformResponse{Success: true, Result: result})
}

func handleValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, litMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	var req ValidateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, ValidateResponse{Success: false, Valid: false, Error: litInvalidRequest + err.Error()})
		return
	}

	if err := hyperex.Validate(req.Expression); err != nil {
		writeJSON(w, ValidateResponse{Success: true, Valid: false, Error: err.Error()})
		return
	}
	writeJSON(w, ValidateResponse{Success: true, Valid: true})
}

func handleExtractVariables(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, litMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	var req ExtractRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, ExtractResponse{Success: false, Error: litInvalidRequest + err.Error()})
		return
	}

	names, err := hyperex.ExtractVariables(req.Expression)
	if err != nil {
		writeJSON(w, ExtractResponse{Success: false, Error: err.Error()})
		return
	}
	writeJSON(w, ExtractResponse{Success: true, Names: names})
}

func handleExtractFunctions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, litMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	var req ExtractRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, ExtractResponse{Success: false, Error: litInvalidRequest + err.Error()})
		return
	}

	names, err := hyperex.ExtractFunctions(req.Expression)
	if err != nil {
		writeJSON(w, ExtractResponse{Success: false, Error: err.Error()})
		return
	}
	writeJSON(w, ExtractResponse{Success: true, Names: names})
}

func handleTypecheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, litMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	var req TypecheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, TypecheckResponse{Success: false, Valid: false, Error: litInvalidRequest + err.Error()})
		return
	}

	if err := hyperex.Validate(req.Expression); err != nil {
		writeJSON(w, TypecheckResponse{Success: true, Valid: false, Error: err.Error()})
		return
	}

	vars, err := hyperex.ExtractVariables(req.Expression)
	if err != nil {
		writeJSON(w, TypecheckResponse{Success: false, Valid: false, Error: err.Error()})
		return
	}
	funcs, err := hyperex.ExtractFunctions(req.Expression)
	if err != nil {
		writeJSON(w, TypecheckResponse{Success: false, Valid: false, Error: err.Error()})
		return
	}

	var unbound []string
	if req.Env != nil {
		for _, name := range vars {
			if _, ok := req.Env[name]; !ok {
				unbound = append(unbound, name)
			}
		}
	}

	// Compile probes bytecode generation and optional type-aware surfaces.
	if _, err := hyperex.Compile(req.Expression); err != nil {
		writeJSON(w, TypecheckResponse{
			Success:   true,
			Valid:     false,
			Error:     err.Error(),
			Variables: vars,
			Functions: funcs,
			Unbound:   unbound,
		})
		return
	}

	writeJSON(w, TypecheckResponse{
		Success:   true,
		Valid:     true,
		Variables: vars,
		Functions: funcs,
		Unbound:   unbound,
	})
}

func handleExamples(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, litMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, GetAllExamples())
}

func handleFunctions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, litMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, GetBuiltinFunctions())
}

func writeJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(data)
}

func getTypeName(v any) string {
	if v == nil {
		return litNil
	}
	switch v.(type) {
	case bool:
		return "bool"
	case int, int8, int16, int32, int64:
		return "int"
	case uint, uint8, uint16, uint32, uint64:
		return "uint"
	case float32, float64:
		return "float"
	case string:
		return "string"
	case []any:
		return "array"
	case map[string]any:
		return exMap
	default:
		return fmt.Sprintf("%T", v)
	}
}
