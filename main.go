// Command astrolift-sample-go is a minimal, single-service Go web fixture for
// the Astrolift validation campaign (spec 42, R3). It serves a small JSON
// star/constellation sample (a nod to Astrolift) plus the common debug
// surface mandated by spec 42 §3.1: GET /health and GET /debug.
//
// Standard library only — no third-party dependencies.
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sort"
	"time"
)

// version is the build commit. It is overridden at link time via
//
//	-ldflags "-X main.version=<sha>"
//
// (see Dockerfile) and otherwise falls back to the GIT_SHA env var, then to
// "dev" for local runs.
var version = ""

// app and kind identify this fixture in the /debug payload. They match the
// repo name and the manifest workload kind (spec 42 §2, R3 = deployment).
const (
	app  = "sample-go"
	kind = "deployment"
)

// startTime anchors the uptime reported by /debug. Captured at process start.
var startTime = time.Now()

// nonSecretEnvKeys are the environment variables we are willing to report as
// "seen" in /debug. We deliberately enumerate an allow-list rather than dump
// os.Environ(), so secrets injected by the platform are never echoed back.
// (spec 42 §3.1: "non-secret keys + presence of binding keys, values
// redacted".) This fixture attaches no managed services, so there are no
// binding keys to add here.
var nonSecretEnvKeys = []string{
	"PORT",
	"LOG_LEVEL",
	"GIT_SHA",
	"HOSTNAME",
	"ENV",
}

// star is one entry in the sample ephemeris catalogue served by /api/stars.
type star struct {
	Name          string `json:"name"`
	Constellation string `json:"constellation"`
	// Magnitude is apparent visual magnitude (lower is brighter).
	Magnitude float64 `json:"magnitude"`
	// DistanceLy is the distance from Earth in light-years.
	DistanceLy float64 `json:"distance_ly"`
}

// catalogue is the static sample data served by /api/stars. Real-but-minimal:
// a handful of well-known navigational stars.
var catalogue = []star{
	{Name: "Sirius", Constellation: "Canis Major", Magnitude: -1.46, DistanceLy: 8.6},
	{Name: "Canopus", Constellation: "Carina", Magnitude: -0.74, DistanceLy: 310},
	{Name: "Arcturus", Constellation: "Boötes", Magnitude: -0.05, DistanceLy: 36.7},
	{Name: "Vega", Constellation: "Lyra", Magnitude: 0.03, DistanceLy: 25},
	{Name: "Polaris", Constellation: "Ursa Minor", Magnitude: 1.98, DistanceLy: 433},
	{Name: "Betelgeuse", Constellation: "Orion", Magnitude: 0.42, DistanceLy: 548},
}

// buildInfo is the "build" block of the /debug payload (spec 42 §3.1).
type buildInfo struct {
	Image   string `json:"image"`
	BuiltAt string `json:"built_at"`
	Commit  string `json:"commit"`
}

// debugInfo is the full /debug diagnostic surface (spec 42 §3.1).
type debugInfo struct {
	App      string            `json:"app"`
	Version  string            `json:"version"`
	Kind     string            `json:"kind"`
	Hostname string            `json:"hostname"`
	UptimeS  int64             `json:"uptime_s"`
	Now      string            `json:"now"`
	EnvSeen  []string          `json:"env_seen"`
	Bindings map[string]string `json:"bindings"`
	Build    buildInfo         `json:"build"`
}

// builtAt is the link-time build timestamp, populated like version via
// -ldflags. Empty for local runs.
var builtAt = ""

func commit() string {
	if version != "" {
		return version
	}
	if sha := os.Getenv("GIT_SHA"); sha != "" {
		return sha
	}
	return "dev"
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		log.Printf("level=error msg=\"failed to encode response\" err=%q", err)
	}
}

// handleStars serves the sample star catalogue (a nod to Astrolift).
func handleStars(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"count": len(catalogue),
		"stars": catalogue,
	})
}

// handleHealth is the readiness/liveness probe target (spec 42 §3.1;
// manifest healthcheck kind="http" value="/health"). Always 200 once the
// process is serving.
func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleDebug serves the diagnostic surface defined in spec 42 §3.1.
func handleDebug(w http.ResponseWriter, r *http.Request) {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	seen := make([]string, 0, len(nonSecretEnvKeys))
	for _, k := range nonSecretEnvKeys {
		if _, ok := os.LookupEnv(k); ok {
			seen = append(seen, k)
		}
	}
	sort.Strings(seen)

	writeJSON(w, http.StatusOK, debugInfo{
		App:      app,
		Version:  commit(),
		Kind:     kind,
		Hostname: hostname,
		UptimeS:  int64(time.Since(startTime).Seconds()),
		Now:      time.Now().UTC().Format(time.RFC3339),
		EnvSeen:  seen,
		// No managed services are attached to this fixture, so there are no
		// bindings to report. Emitted as an empty object (not null) so the
		// shape is stable for consumers.
		Bindings: map[string]string{},
		Build: buildInfo{
			Image:   "docker.io/calliopeai/astrolift-sample-go",
			BuiltAt: builtAt,
			Commit:  commit(),
		},
	})
}

func port() string {
	if p := os.Getenv("PORT"); p != "" {
		return p
	}
	return "8080"
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/stars", handleStars)
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/debug", handleDebug)

	addr := ":" + port()
	hostname, _ := os.Hostname()

	// Structured startup banner (spec 42 §3.1): app, version, detected
	// bindings, listen port — logged at boot so logs are immediately useful.
	log.Printf(
		"level=info msg=\"starting astrolift-sample-go\" app=%s version=%s kind=%s hostname=%s addr=%s bindings=none",
		app, commit(), kind, hostname, addr,
	)

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("level=error msg=\"server exited\" err=%q", err)
	}
}
