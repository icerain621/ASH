package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/ash-repwiki/ash/internal/config"
	"github.com/ash-repwiki/ash/internal/doctor"
	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/runs"
	"github.com/ash-repwiki/ash/internal/rules"
	"github.com/ash-repwiki/ash/internal/store"
	"github.com/ash-repwiki/ash/internal/toolbus"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "doctor":
		runDoctor(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: ash <command>\n\nCommands:\n  doctor --suite TR0 [--format json|md] [--out path]\n")
}

func runDoctor(args []string) {
	fs := flag.NewFlagSet("doctor", flag.ExitOnError)
	suite := fs.String("suite", "TR0", "validation suite: TR0|ALL")
	format := fs.String("format", "json", "output format: json|md")
	out := fs.String("out", "", "write report to file")
	_ = fs.Parse(args)

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

	if rep.Summary.Fail > 0 {
		os.Exit(1)
	}
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
