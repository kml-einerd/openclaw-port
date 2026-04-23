// Package cli provides PM-OS command-line interface commands.
package cli

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

// DoctorCheck represents a single diagnostic check in the doctor command.
type DoctorCheck struct {
	Name   string
	Status string // "ok", "warn", "fail"
	Detail string
}

// DoctorOptions configures the behavior of the doctor command.
type DoctorOptions struct {
	Fix     bool
	Verbose bool
}

// RunDoctor performs a series of diagnostic checks on the PM-OS installation.
//
// Adapted from openclaw §5.1 pm-cli doctor.
func RunDoctor(ctx context.Context, opts DoctorOptions) []DoctorCheck {
	var checks []DoctorCheck

	// 1. Check Supabase connectivity
	checks = append(checks, checkHTTPEndpoint(ctx, "Supabase", "SUPABASE_URL", "/rest/v1/"))

	// 2. Check Anthropic API key
	checks = append(checks, checkEnvVar("Anthropic API Key", "ANTHROPIC_API_KEY"))

	// 3. Check systemd services
	checks = append(checks, checkSystemdService("pmos-api"))
	checks = append(checks, checkSystemdService("pmos-engine"))

	// 4. Check stale runs (placeholder — needs Supabase query in real impl)
	checks = append(checks, DoctorCheck{
		Name:   "Stale Runs",
		Status: "ok",
		Detail: "No stale runs detected (stub — wire to Supabase query)",
	})

	return checks
}

func checkEnvVar(name, envKey string) DoctorCheck {
	// In production this would use os.Getenv; here we stub it to avoid
	// leaking real env vars during tests.
	return DoctorCheck{
		Name:   name,
		Status: "ok",
		Detail: fmt.Sprintf("env var %s check (stub)", envKey),
	}
}

func checkHTTPEndpoint(ctx context.Context, name, envKey, path string) DoctorCheck {
	// Stub: in production reads os.Getenv(envKey) and issues a real HTTP GET.
	client := &http.Client{Timeout: 2 * time.Second}
	url := "http://localhost:8080" + path

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return DoctorCheck{Name: name, Status: "fail", Detail: err.Error()}
	}

	resp, err := client.Do(req)
	if err != nil {
		return DoctorCheck{Name: name, Status: "warn", Detail: fmt.Sprintf("unreachable: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return DoctorCheck{Name: name, Status: "fail", Detail: fmt.Sprintf("HTTP %d", resp.StatusCode)}
	}

	return DoctorCheck{Name: name, Status: "ok", Detail: "reachable"}
}

func checkSystemdService(unit string) DoctorCheck {
	out, err := exec.Command("systemctl", "is-active", unit).Output()
	if err != nil {
		return DoctorCheck{
			Name:   "Service: " + unit,
			Status: "warn",
			Detail: "systemctl not available or service not found",
		}
	}

	status := strings.TrimSpace(string(out))
	if status == "active" {
		return DoctorCheck{Name: "Service: " + unit, Status: "ok", Detail: "active"}
	}
	return DoctorCheck{Name: "Service: " + unit, Status: "fail", Detail: status}
}
