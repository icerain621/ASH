package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/ash-repwiki/ash/internal/agenttpl"
	"github.com/ash-repwiki/ash/internal/config"
	"github.com/ash-repwiki/ash/internal/review"
	"github.com/ash-repwiki/ash/internal/review/passk"
	"github.com/ash-repwiki/ash/internal/store"
)

func runReview(args []string) {
	if len(args) < 1 {
		log.Fatal(reviewUsage())
	}
	switch args[0] {
	case "list":
		runReviewList(args[1:])
	case "decide":
		runReviewDecide(args[1:])
	case "passk":
		runReviewPassK(args[1:])
	case "templates":
		runReviewTemplates(args[1:])
	case "eval":
		runReviewEval(args[1:])
	default:
		log.Fatal(reviewUsage())
	}
}

func reviewUsage() string {
	return "review requires list|decide|passk|templates|eval\n" +
		"  ash review list [--kind agent_template] [--space local] [--format json|md]\n" +
		"  ash review decide <reviewId> --decision approve|reject [--reason text] [--space local]\n" +
		"  ash review passk --target tpl.react [--k 2]\n" +
		"  ash review templates [--space local] [--format json|md]\n" +
		"  ash review eval [--space local]  (F05 memory counters)\n" +
		"Quality checks belong here, not under ash doctor (F07)."
}

func openReviewDB() *store.DB {
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
			log.Fatalf("postgres review session: %v", err)
		}
	}
	installCLIApprovedLoader(db)
	return db
}

func installCLIApprovedLoader(db *store.DB) {
	agenttpl.SetApprovedLoader(func(id string) (agenttpl.Manifest, bool) {
		row, err := db.GetApprovedAgentTemplate("local", id)
		if err != nil || row == nil {
			return agenttpl.Manifest{}, false
		}
		m, err := agenttpl.ManifestFromJSON(row.ManifestJSON)
		if err != nil {
			return agenttpl.Manifest{}, false
		}
		return m, true
	})
}

func runReviewList(args []string) {
	fs := flag.NewFlagSet("review list", flag.ExitOnError)
	kind := fs.String("kind", "", "optional kind filter")
	space := fs.String("space", "local", "space id")
	format := fs.String("format", "json", "json|md")
	_ = fs.Parse(args)
	db := openReviewDB()
	rows, err := db.ListPendingReviewItems(*space, *kind)
	if err != nil {
		log.Fatalf("list: %v", err)
	}
	if *format == "md" {
		fmt.Println(formatReviewListMD(rows))
		return
	}
	b, err := json.MarshalIndent(map[string]any{"items": rows}, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(b))
}

func formatReviewListMD(rows []store.ReviewItemRow) string {
	var b strings.Builder
	b.WriteString("# ASH Review Queue\n\n")
	if len(rows) == 0 {
		b.WriteString("_empty_\n")
		return b.String()
	}
	for _, row := range rows {
		fmt.Fprintf(&b, "- `%s` · %s · %s · %s\n", row.ID, row.Kind, row.TargetID, row.Status)
	}
	return b.String()
}

func runReviewDecide(args []string) {
	fs := flag.NewFlagSet("review decide", flag.ExitOnError)
	decision := fs.String("decision", "", "approve|reject")
	reason := fs.String("reason", "cli review", "decision reason")
	space := fs.String("space", "local", "space id")
	_ = fs.Parse(args)
	if fs.NArg() < 1 || strings.TrimSpace(*decision) == "" {
		log.Fatal("usage: ash review decide <reviewId> --decision approve|reject")
	}
	db := openReviewDB()
	row, err := db.DecideReviewItem(fs.Arg(0), *decision, "cli", *reason)
	if err != nil {
		log.Fatalf("decide: %v", err)
	}
	if row.Status == "approved" && row.Kind == "agent_template" {
		_, _ = agenttpl.ApproveTemplate(row.TargetID)
		_, _ = db.ApproveAgentTemplate(*space, row.TargetID, "cli")
	}
	b, _ := json.MarshalIndent(row, "", "  ")
	fmt.Println(string(b))
}

func runReviewPassK(args []string) {
	fs := flag.NewFlagSet("review passk", flag.ExitOnError)
	target := fs.String("target", "tpl.react", "template or check target id")
	k := fs.Int("k", 2, "sample count")
	_ = fs.Parse(args)
	_ = openReviewDB() // installs approved loader
	res, err := passk.Run(passk.Config{
		K: *k,
		Check: func(i int) error {
			_, err := agenttpl.GetProduction(strings.TrimSpace(*target))
			return err
		},
	})
	if err != nil {
		log.Fatalf("passk: %v", err)
	}
	b, _ := json.MarshalIndent(res, "", "  ")
	fmt.Println(string(b))
	if !res.AllPass {
		os.Exit(1)
	}
}

func runReviewEval(args []string) {
	fs := flag.NewFlagSet("review eval", flag.ExitOnError)
	space := fs.String("space", "local", "space id")
	_ = fs.Parse(args)
	db := openReviewDB()
	counts, err := review.LoadMemoryEvalCounts(db, *space, 5000)
	if err != nil {
		log.Fatalf("eval: %v", err)
	}
	b, _ := json.MarshalIndent(counts, "", "  ")
	fmt.Println(string(b))
}

func runReviewTemplates(args []string) {
	fs := flag.NewFlagSet("review templates", flag.ExitOnError)
	space := fs.String("space", "local", "space id")
	format := fs.String("format", "json", "json|md")
	_ = fs.Parse(args)
	db := openReviewDB()
	entries, err := agenttpl.ListEntries()
	if err != nil {
		log.Fatalf("templates: %v", err)
	}
	rows, _ := db.ListAgentTemplateVersions(*space)
	seen := map[string]struct{}{}
	for _, e := range entries.Items {
		seen[e.Manifest.ID] = struct{}{}
	}
	for _, row := range rows {
		if _, ok := seen[row.TemplateID]; ok {
			continue
		}
		m, err := agenttpl.ManifestFromJSON(row.ManifestJSON)
		if err != nil {
			m = agenttpl.Manifest{ID: row.TemplateID, Version: row.Version}
		}
		entries.Items = append(entries.Items, agenttpl.Entry{Manifest: m, Status: row.Status})
	}
	if *format == "md" {
		var b strings.Builder
		b.WriteString("# ASH Agent Templates\n\n")
		for _, e := range entries.Items {
			bind := "no"
			if e.Status == agenttpl.StatusApproved {
				bind = "yes"
			}
			fmt.Fprintf(&b, "- `%s` · %s · bindable=%s\n", e.Manifest.ID, e.Status, bind)
		}
		fmt.Print(b.String())
		return
	}
	b, _ := json.MarshalIndent(entries, "", "  ")
	fmt.Println(string(b))
}

// doctorQualityMigrated reports the F07 migration message when a quality suite is requested via doctor.
func doctorQualityMigrated(suite string) (string, bool) {
	if strings.EqualFold(strings.TrimSpace(suite), "QUALITY") {
		return "QUALITY moved out of ash doctor (F07). Use: ash review passk --target <id> (or console 评审管控 → 组件闸 / HTTP /api/v1/passk/run).", true
	}
	return "", false
}
