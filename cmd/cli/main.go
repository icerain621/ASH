package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/ash-repwiki/ash/internal/agentexec"
	"github.com/ash-repwiki/ash/internal/artifacts"
	"github.com/ash-repwiki/ash/internal/config"
	"github.com/ash-repwiki/ash/internal/doctor"
	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/goal"
	"github.com/ash-repwiki/ash/internal/pluginabi"
	"github.com/ash-repwiki/ash/internal/rules"
	"github.com/ash-repwiki/ash/internal/runs"
	"github.com/ash-repwiki/ash/internal/store"
	"github.com/ash-repwiki/ash/internal/toolbus"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "run":
		runScenario(os.Args[2:])
	case "quest":
		runQuest(os.Args[2:])
	case "replay":
		runReplay(os.Args[2:])
	case "cancel":
		runCancel(os.Args[2:])
	case "doctor":
		runDoctor(os.Args[2:])
	case "migrate":
		runMigrate(os.Args[2:])
	case "plugin-sign":
		runPluginSign(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: ash <command>\n\nCommands:\n  run --issue text [--repo .] [--scenario feature_delivery] [--version 1.0.0] [--agent execgo_codex|static]\n  quest \"goal text\" [--repo .] [--yes] [--agent execgo_codex|static]  (Goal→Plan→Run)\n  replay <runId> [--mode exact|latest_memory] [--agent execgo_codex|static]\n  cancel <runId> [--agent execgo_codex|static]\n  doctor --suite TR0|TR1|TR2|TR3|M2|M3|M4|M5|ALL [--format json|md] [--require M3-04,M3-06] [--out path] [--agent execgo_codex|static]\n  migrate plan|copy|verify|sync|dual-write ...  (sqlite→postgres migration)\n  plugin-sign --name n --version v --endpoint host:port [--key env|literal]  (HMAC plugin signature)\n")
}

func runPluginSign(args []string) {
	fs := flag.NewFlagSet("plugin-sign", flag.ExitOnError)
	name := fs.String("name", "", "plugin name")
	version := fs.String("version", "", "plugin version")
	protocol := fs.String("protocol", "grpc", "protocol")
	abi := fs.String("abi", pluginabi.CurrentABI, "ABI version")
	endpoint := fs.String("endpoint", "", "plugin endpoint")
	key := fs.String("key", "", "HMAC key (default: ASH_PLUGIN_SIGNING_KEY)")
	capability := fs.Bool("capability", false, "print ash.sign.hmac=<hex> instead of bare hex")
	_ = fs.Parse(args)
	if strings.TrimSpace(*name) == "" || strings.TrimSpace(*version) == "" || strings.TrimSpace(*endpoint) == "" {
		log.Fatal("--name, --version, and --endpoint are required")
	}
	signKey := strings.TrimSpace(*key)
	if signKey == "" {
		signKey = pluginabi.SigningKey()
	}
	if signKey == "" {
		log.Fatal("signing key required: pass --key or set ASH_PLUGIN_SIGNING_KEY")
	}
	sig := pluginabi.SignHMAC(signKey, *name, *version, *protocol, *abi, *endpoint)
	if *capability {
		fmt.Println(pluginabi.CapabilitySignPrefix + sig)
		return
	}
	fmt.Println(sig)
}

func runScenario(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	scenario := fs.String("scenario", "feature_delivery", "scenario name")
	version := fs.String("version", "1.0.0", "scenario version")
	issue := fs.String("issue", "", "issue/spec text")
	repo := fs.String("repo", ".", "repository root")
	policy := fs.String("policy", "", "policy profile")
	agent := fs.String("agent", "execgo_codex", "agent executor: execgo_codex|static")
	format := fs.String("format", "json", "output format: json|md")
	_ = fs.Parse(args)
	if *issue == "" {
		log.Fatal("--issue is required")
	}

	runsSvc := buildRunsService(*agent)
	resp, err := runsSvc.Create(runs.CreateRequest{
		Scenario:      runs.ScenarioRef{Name: *scenario, ScenarioVersion: *version},
		PolicyProfile: *policy,
		Repo:          &runs.RepoRef{Root: *repo},
		Inputs: map[string]any{
			"issueOrSpec": *issue,
			"repoRoot":    *repo,
		},
	})
	if err != nil {
		log.Fatalf("run failed: %v", err)
	}
	emitRunResult(runsSvc, resp.RunID, resp.TraceID, *format)
}

func runQuest(args []string) {
	fs := flag.NewFlagSet("quest", flag.ExitOnError)
	repo := fs.String("repo", ".", "repository root")
	yes := fs.Bool("yes", false, "auto-approve plan and start run")
	agent := fs.String("agent", "static", "agent executor: execgo_codex|static")
	format := fs.String("format", "json", "output format: json|md")
	_ = fs.Parse(args)
	goalText := strings.TrimSpace(strings.Join(fs.Args(), " "))
	if goalText == "" {
		log.Fatal("quest requires a goal string, e.g. ash quest \"Add dark mode\"")
	}
	cfg := config.Load()
	db, err := store.Open(cfg.DataDir)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	scenariosDir := resolveScenariosDir(cfg.ScenariosDir)
	loader := rules.NewLoader(scenariosDir)
	if err := loader.LoadDir(); err != nil {
		log.Fatalf("load scenarios: %v", err)
	}
	ev := events.NewService(db)
	runsSvc := runs.NewService(db, ev, loader, toolbus.DefaultBus())
	if *agent == "static" {
		runsSvc.WithAgentExecutor(agentexec.StaticExecutor{})
	}
	goalSvc := goal.NewService(db, loader, runsSvc, ev)
	plan, err := goalSvc.FromGoal(goal.FromGoalRequest{
		Goal: goalText, RepoRoot: *repo, SpaceID: "local", CreatedBy: "cli", AutoApprove: *yes,
	})
	if err != nil && plan == nil {
		log.Fatalf("quest failed: %v", err)
	}
	if !*yes {
		b, _ := json.MarshalIndent(plan, "", "  ")
		fmt.Println(string(b))
		fmt.Fprintf(os.Stderr, "Plan %s is draft. Re-run with --yes to approve, or POST /runs/plans/%s/approve\n", plan.ID, plan.ID)
		return
	}
	if plan.RunID == "" {
		log.Fatal("auto-approve did not produce runId")
	}
	emitRunResult(runsSvc, plan.RunID, plan.TraceID, *format)
}

func runReplay(args []string) {
	fs := flag.NewFlagSet("replay", flag.ExitOnError)
	mode := fs.String("mode", "exact", "replay mode: exact|latest_memory")
	agent := fs.String("agent", "execgo_codex", "agent executor: execgo_codex|static")
	format := fs.String("format", "json", "output format: json|md")
	_ = fs.Parse(args)
	if fs.NArg() < 1 {
		log.Fatal("replay requires <runId>")
	}
	sourceRunID := fs.Arg(0)
	runsSvc := buildRunsService(*agent)
	resp, err := runsSvc.Replay(sourceRunID, runs.ReplayRequest{Mode: *mode})
	if err != nil {
		log.Fatalf("replay failed: %v", err)
	}
	emitRunResult(runsSvc, resp.RunID, resp.TraceID, *format)
}

func runCancel(args []string) {
	fs := flag.NewFlagSet("cancel", flag.ExitOnError)
	agent := fs.String("agent", "execgo_codex", "agent executor: execgo_codex|static")
	_ = fs.Parse(args)
	if fs.NArg() < 1 {
		log.Fatal("cancel requires <runId>")
	}
	runsSvc := buildRunsService(*agent)
	resp, err := runsSvc.Cancel(fs.Arg(0))
	if err != nil {
		log.Fatalf("cancel failed: %v", err)
	}
	b, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Println(string(b))
}

func runDoctor(args []string) {
	fs := flag.NewFlagSet("doctor", flag.ExitOnError)
	suite := fs.String("suite", "TR0", "validation suite: TR0|TR1|TR2|TR3|M2|ALL")
	format := fs.String("format", "json", "output format: json|md")
	out := fs.String("out", "", "write report to file")
	agent := fs.String("agent", "execgo_codex", "agent executor: execgo_codex|static")
	require := fs.String("require", "", "comma-separated case IDs that must pass without skip evidence")
	_ = fs.Parse(args)

	cfg := config.Load()
	dbURL := strings.TrimSpace(os.Getenv("ASH_DATABASE_URL"))
	if dbURL == "" {
		dbURL = store.RuntimeDatabaseURL()
	}
	db, err := store.OpenWithDatabaseURL(cfg.DataDir, dbURL)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	if db.Dialect() == "postgres" {
		if err := db.Exec("SET row_security = off").Error; err != nil {
			log.Fatalf("postgres doctor session: %v", err)
		}
	}

	scenariosDir := resolveScenariosDir(cfg.ScenariosDir)
	loader := rules.NewLoader(scenariosDir)
	if err := loader.LoadDir(); err != nil {
		log.Fatalf("load scenarios: %v", err)
	}

	ev := events.NewService(db)
	runsSvc := runs.NewService(db, ev, loader, toolbus.DefaultBus())
	if *agent == "static" {
		runsSvc.WithAgentExecutor(agentexec.StaticExecutor{})
	}
	doc := doctor.NewService(runsSvc, ev, loader, db.DataDir())

	rep, err := doc.RunSuite(*suite)
	if err != nil {
		log.Fatalf("doctor: %v", err)
	}

	var payload []byte
	if *format == "md" {
		payload = []byte(formatReportMD(rep))
	} else {
		payload, err = json.MarshalIndent(rep, "", "  ")
		if err != nil {
			log.Fatalf("marshal: %v", err)
		}
	}

	if *out != "" {
		if err := os.WriteFile(*out, payload, 0o644); err != nil {
			log.Fatalf("write: %v", err)
		}
		fmt.Printf("report written to %s (pass=%d fail=%d)\n", *out, rep.Summary.Pass, rep.Summary.Fail)
	} else {
		fmt.Println(string(payload))
	}

	if strings.TrimSpace(*require) != "" {
		ids := strings.Split(*require, ",")
		for i := range ids {
			ids[i] = strings.TrimSpace(ids[i])
		}
		if err := doctor.RequireCases(rep, ids, true); err != nil {
			log.Fatal(err)
		}
		return
	}
	if rep.Summary.Fail > 0 {
		os.Exit(1)
	}
}

func buildRunsService(agentName string) *runs.Service {
	cfg := config.Load()
	db, err := store.Open(cfg.DataDir)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	scenariosDir := resolveScenariosDir(cfg.ScenariosDir)
	loader := rules.NewLoader(scenariosDir)
	if err := loader.LoadDir(); err != nil {
		log.Fatalf("load scenarios: %v", err)
	}
	ev := events.NewService(db)
	svc := runs.NewService(db, ev, loader, toolbus.DefaultBus())
	if agentName == "static" {
		svc.WithAgentExecutor(agentexec.StaticExecutor{})
	}
	return svc
}

func emitRunResult(runsSvc *runs.Service, runID, traceID, format string) {
	manifest, _ := runsSvc.Artifacts(runID)
	timeline, _ := runsSvc.Events().ListAfter(runID, 0, 500)
	artifacts := artifactResults(runsSvc, runID, manifest)
	checkpoints := checkpointResults(runsSvc, runID)
	if format == "md" {
		fmt.Printf("# ASH Run\n\n- runId: `%s`\n- traceId: `%s`\n", runID, traceID)
		if len(artifacts) > 0 {
			fmt.Println("\n## Artifacts")
			for _, a := range artifacts {
				fmt.Printf("- `%s`: %s (%s)\n", a.Type, a.URI, a.Digest)
				if a.AccessURL != "" {
					fmt.Printf("  - access: %s\n", a.AccessURL)
				}
			}
		}
		if len(checkpoints) > 0 {
			fmt.Println("\n## Checkpoints")
			for _, c := range checkpoints {
				fmt.Printf("- `%s`: %s (%s)\n", c.ID, c.URI, c.SnapshotDigest)
				if c.AccessURL != "" {
					fmt.Printf("  - access: %s\n", c.AccessURL)
				}
			}
		}
		if len(timeline) > 0 {
			fmt.Println("\n## Events")
			for _, ev := range timeline {
				fmt.Printf("- #%d `%s` %s\n", ev.Seq, ev.Type, ev.Severity)
			}
		}
		return
	}
	payload := map[string]any{
		"runId": runID, "traceId": traceID,
		"artifacts":   artifacts,
		"checkpoints": checkpoints,
		"events":      timeline,
	}
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		log.Fatalf("marshal result: %v", err)
	}
	fmt.Println(string(b))
}

type cliArtifact struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	URI         string `json:"uri"`
	AccessURL   string `json:"accessUrl,omitempty"`
	Digest      string `json:"digest"`
	ContentType string `json:"contentType,omitempty"`
	SizeBytes   int64  `json:"sizeBytes,omitempty"`
}

type cliCheckpoint struct {
	ID             string `json:"id"`
	StepID         string `json:"stepId"`
	URI            string `json:"uri"`
	AccessURL      string `json:"accessUrl,omitempty"`
	SnapshotDigest string `json:"snapshotDigest"`
	ContentType    string `json:"contentType,omitempty"`
	SizeBytes      int64  `json:"sizeBytes,omitempty"`
	Strategy       string `json:"strategy,omitempty"`
}

func artifactResults(runsSvc *runs.Service, runID string, manifest *artifacts.Manifest) []cliArtifact {
	if manifest == nil {
		return nil
	}
	out := make([]cliArtifact, 0, len(manifest.Artifacts))
	for _, a := range manifest.Artifacts {
		item := cliArtifact{
			Type: a.Type, Name: a.Name, URI: a.URI, Digest: a.Digest,
			ContentType: a.ContentType, SizeBytes: a.SizeBytes,
		}
		if access, err := runsSvc.ArtifactAccess(runID, a.Name, 0); err == nil && access != nil {
			item.AccessURL = access.SignedURL
			if access.URI != "" {
				item.URI = access.URI
			}
		}
		out = append(out, item)
	}
	return out
}

func checkpointResults(runsSvc *runs.Service, runID string) []cliCheckpoint {
	rows, err := runsSvc.Checkpoints(runID)
	if err != nil || len(rows) == 0 {
		return nil
	}
	out := make([]cliCheckpoint, 0, len(rows))
	for _, c := range rows {
		item := cliCheckpoint{
			ID: c.ID, StepID: c.StepID, URI: c.URI, SnapshotDigest: c.SnapshotDigest,
			ContentType: c.ContentType, SizeBytes: c.SizeBytes, Strategy: c.Strategy,
		}
		if access, err := runsSvc.CheckpointAccess(runID, c.ID, 0); err == nil && access != nil {
			item.AccessURL = access.SignedURL
			if access.URI != "" {
				item.URI = access.URI
			}
		}
		out = append(out, item)
	}
	return out
}

func resolveScenariosDir(dir string) string {
	if filepath.IsAbs(dir) {
		return dir
	}
	if wd, err := os.Getwd(); err == nil {
		return filepath.Join(wd, dir)
	}
	return dir
}

func formatReportMD(rep *doctor.Report) string {
	var b string
	b += fmt.Sprintf("# ASH Doctor Report — %s\n\n", rep.Suite)
	b += fmt.Sprintf("- Pass: **%d** | Fail: **%d**\n\n", rep.Summary.Pass, rep.Summary.Fail)
	for _, r := range rep.Results {
		icon := "✅"
		if r.Status != "pass" {
			icon = "❌"
		}
		b += fmt.Sprintf("## %s %s\n\n", icon, r.ID)
		if r.RunID != "" {
			b += fmt.Sprintf("- runId: `%s`\n", r.RunID)
		}
		if r.Message != "" {
			b += fmt.Sprintf("- message: %s\n", r.Message)
		}
		for _, e := range r.Evidence {
			b += fmt.Sprintf("- evidence: %s `%s`", e.Kind, e.Ref)
			if e.Digest != "" {
				b += fmt.Sprintf(" (%s)", e.Digest)
			}
			b += "\n"
		}
		b += "\n"
	}
	return b
}
