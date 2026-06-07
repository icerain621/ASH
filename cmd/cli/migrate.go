package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/ash-repwiki/ash/internal/config"
	"github.com/ash-repwiki/ash/internal/store"
	"github.com/ash-repwiki/ash/internal/store/sqlmigrations"
)

func runMigrate(args []string) {
	if len(args) == 0 {
		migrateUsage()
		os.Exit(2)
	}
	switch args[0] {
	case "plan":
		runMigratePlan(args[1:])
	case "copy":
		runMigrateCopy(args[1:])
	case "verify":
		runMigrateVerify(args[1:])
	case "sync":
		runMigrateSync(args[1:])
	case "dual-write":
		runMigrateDualWrite(args[1:])
	case "schema":
		runMigrateSchema(args[1:])
	default:
		migrateUsage()
		os.Exit(2)
	}
}

func migrateUsage() {
	fmt.Fprintf(os.Stderr, `Usage: ash migrate <subcommand>

Subcommands:
  plan       Show sqlite/postgres table row counts
  copy       Upsert all rows from sqlite into postgres
  verify     Fail when per-table row counts differ
  sync       Incremental upsert (uses .ash/migration/sync-state.json)
  dual-write enable|disable|status|sync
  schema     Apply or inspect golang-migrate SQL revisions (Postgres)

Flags (plan/copy/verify/sync):
  --data-dir <path>       ASH data dir (default: .ash)
  --sqlite <path>         sqlite file (default: <data-dir>/ash.db)
  --postgres <url>        postgres URL (or ASH_DATABASE_URL / dual-write config)

Examples:
  ash migrate plan --postgres "$ASH_DATABASE_URL"
  ash migrate copy --postgres postgres://ash:ash@127.0.0.1:5432/ash?sslmode=disable
  ash migrate dual-write enable --postgres "$ASH_DATABASE_URL"
  export ASH_DUAL_WRITE_POSTGRES_URL="$ASH_DATABASE_URL"  # runtime mirror on worker
`)
}

type migrateFlags struct {
	dataDir  string
	sqlite   string
	postgres string
	batch    int
	dryRun   bool
	tables   string
	format   string
}

func parseMigrateFlags(fs *flag.FlagSet, args []string) migrateFlags {
	cfg := config.Load()
	flags := migrateFlags{
		dataDir: cfg.DataDir,
		sqlite:  store.DefaultSQLitePath(cfg.DataDir),
	}
	fs.StringVar(&flags.dataDir, "data-dir", flags.dataDir, "ASH data directory")
	fs.StringVar(&flags.sqlite, "sqlite", flags.sqlite, "sqlite database file")
	fs.StringVar(&flags.postgres, "postgres", "", "postgres URL")
	fs.IntVar(&flags.batch, "batch", 200, "copy batch size")
	fs.BoolVar(&flags.dryRun, "dry-run", false, "count rows only")
	fs.StringVar(&flags.tables, "tables", "", "comma-separated table allowlist")
	fs.StringVar(&flags.format, "format", "json", "output format: json|md")
	_ = fs.Parse(args)
	if flags.postgres == "" {
		flags.postgres = strings.TrimSpace(cfg.DatabaseURL)
	}
	if flags.postgres == "" {
		if dwc, err := store.LoadDualWriteConfig(flags.dataDir); err == nil && dwc.Enabled {
			flags.postgres = dwc.PostgresURL
		}
	}
	return flags
}

func openMigrator(flags migrateFlags) *store.Migrator {
	if strings.TrimSpace(flags.postgres) == "" {
		log.Fatal("--postgres or ASH_DATABASE_URL is required")
	}
	m, err := store.NewMigrator(flags.dataDir, flags.sqlite, flags.postgres)
	if err != nil {
		log.Fatalf("open migrator: %v", err)
	}
	return m
}

func runMigratePlan(args []string) {
	fs := flag.NewFlagSet("migrate plan", flag.ExitOnError)
	flags := parseMigrateFlags(fs, args)
	m := openMigrator(flags)
	defer m.Close()

	plan, err := m.Plan()
	if err != nil {
		log.Fatalf("plan: %v", err)
	}
	emitMigratePayload(flags.format, plan, formatMigratePlanMD)
}

func runMigrateCopy(args []string) {
	fs := flag.NewFlagSet("migrate copy", flag.ExitOnError)
	flags := parseMigrateFlags(fs, args)
	m := openMigrator(flags)
	defer m.Close()

	report, err := m.Copy(store.CopyOptions{
		BatchSize: flags.batch,
		DryRun:    flags.dryRun,
		Tables:    splitCSV(flags.tables),
	})
	if err != nil {
		log.Fatalf("copy: %v", err)
	}
	emitMigratePayload(flags.format, report, formatCopyReportMD)
}

func runMigrateVerify(args []string) {
	fs := flag.NewFlagSet("migrate verify", flag.ExitOnError)
	flags := parseMigrateFlags(fs, args)
	m := openMigrator(flags)
	defer m.Close()

	plan, err := m.Verify()
	if err != nil {
		emitMigratePayload(flags.format, plan, formatMigratePlanMD)
		log.Fatalf("verify: %v", err)
	}
	emitMigratePayload(flags.format, plan, formatMigratePlanMD)
}

func runMigrateSync(args []string) {
	fs := flag.NewFlagSet("migrate sync", flag.ExitOnError)
	flags := parseMigrateFlags(fs, args)
	m := openMigrator(flags)
	defer m.Close()

	report, err := m.Sync(flags.dataDir, store.CopyOptions{
		BatchSize: flags.batch,
		DryRun:    flags.dryRun,
		Tables:    splitCSV(flags.tables),
	})
	if err != nil {
		log.Fatalf("sync: %v", err)
	}
	emitMigratePayload(flags.format, report, formatCopyReportMD)
}

func runMigrateDualWrite(args []string) {
	if len(args) == 0 {
		log.Fatal("dual-write requires enable|disable|status|sync")
	}
	switch args[0] {
	case "enable":
		runDualWriteEnable(args[1:])
	case "disable":
		runDualWriteDisable(args[1:])
	case "status":
		runDualWriteStatus(args[1:])
	case "sync":
		runMigrateSync(args[1:])
	default:
		log.Fatalf("unknown dual-write subcommand %q", args[0])
	}
}

func runDualWriteEnable(args []string) {
	fs := flag.NewFlagSet("migrate dual-write enable", flag.ExitOnError)
	flags := parseMigrateFlags(fs, args)
	if strings.TrimSpace(flags.postgres) == "" {
		log.Fatal("--postgres is required")
	}
	cfg := &store.DualWriteConfig{
		Enabled: true, PostgresURL: flags.postgres, SQLitePath: flags.sqlite,
		EnabledAt: time.Now().UTC(),
	}
	if err := store.SaveDualWriteConfig(flags.dataDir, cfg); err != nil {
		log.Fatalf("save dual-write config: %v", err)
	}
	fmt.Printf("dual-write enabled: %s\n", store.DualWriteConfigPath(flags.dataDir))
	fmt.Println("Restart Worker (sqlite primary) to load dual-write from config.")
	fmt.Printf("Optional override: export ASH_DUAL_WRITE_POSTGRES_URL=%q\n", flags.postgres)
}

func runDualWriteDisable(args []string) {
	fs := flag.NewFlagSet("migrate dual-write disable", flag.ExitOnError)
	flags := parseMigrateFlags(fs, args)
	if err := store.SaveDualWriteConfig(flags.dataDir, &store.DualWriteConfig{Enabled: false}); err != nil {
		log.Fatalf("disable: %v", err)
	}
	fmt.Println("dual-write config disabled; unset ASH_DUAL_WRITE_POSTGRES_URL on workers")
}

func runDualWriteStatus(args []string) {
	fs := flag.NewFlagSet("migrate dual-write status", flag.ExitOnError)
	flags := parseMigrateFlags(fs, args)
	cfg, err := store.LoadDualWriteConfig(flags.dataDir)
	if err != nil {
		log.Fatalf("status: %v", err)
	}
	state, err := store.LoadSyncState(flags.dataDir)
	if err != nil {
		log.Fatalf("sync state: %v", err)
	}
	payload := map[string]any{
		"dualWrite": cfg,
		"syncState": state,
		"runtimeEnv": map[string]any{
			"ASH_DUAL_WRITE_POSTGRES_URL": os.Getenv("ASH_DUAL_WRITE_POSTGRES_URL") != "",
		},
	}
	emitMigratePayload(flags.format, payload, func(v any) string {
		m := v.(map[string]any)
		return fmt.Sprintf("# Dual-write status\n\n```json\n%s\n```\n", mustJSON(m))
	})
}

func emitMigratePayload(format string, payload any, mdFn func(any) string) {
	if format == "md" {
		fmt.Println(mdFn(payload))
		return
	}
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		log.Fatalf("marshal: %v", err)
	}
	fmt.Println(string(b))
}

func formatMigratePlanMD(v any) string {
	plan := v.(*store.MigrationPlan)
	var b strings.Builder
	fmt.Fprintf(&b, "# Migration Plan\n\n- source: `%s` (%s)\n- target: `%s` (%s)\n- ready: %v\n\n",
		plan.SourceDSN, plan.SourceDialect, plan.TargetDSN, plan.TargetDialect, plan.Ready)
	b.WriteString("| table | source | target | match |\n|---|---:|---:|:---:|\n")
	for _, row := range plan.Tables {
		match := "yes"
		if !row.Match {
			match = "no"
		}
		fmt.Fprintf(&b, "| %s | %d | %d | %s |\n", row.Table, row.SourceRows, row.TargetRows, match)
	}
	return b.String()
}

func formatCopyReportMD(v any) string {
	report := v.(*store.CopyReport)
	var b strings.Builder
	fmt.Fprintf(&b, "# Migration Copy\n\n- total: **%d** rows\n- incremental: %v\n- started: %s\n- finished: %s\n\n",
		report.TotalCopied, report.Incremental, report.StartedAt.Format(time.RFC3339), report.FinishedAt.Format(time.RFC3339))
	for _, row := range report.Tables {
		fmt.Fprintf(&b, "- `%s`: %d", row.Table, row.Copied)
		if row.DryRun {
			b.WriteString(" (dry-run)")
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func runMigrateSchema(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: ash migrate schema <up|down|version> [--postgres <url>]\n")
		os.Exit(2)
	}
	fs := flag.NewFlagSet("migrate schema", flag.ExitOnError)
	postgres := fs.String("postgres", "", "postgres URL (default ASH_DATABASE_URL)")
	_ = fs.Parse(args[1:])
	sub := args[0]
	dsn := strings.TrimSpace(*postgres)
	if dsn == "" {
		dsn = strings.TrimSpace(os.Getenv("ASH_DATABASE_URL"))
	}
	if dsn == "" {
		log.Fatal("postgres URL required (--postgres or ASH_DATABASE_URL)")
	}
	switch sub {
	case "up":
		v, err := sqlmigrations.ApplyPostgres(dsn)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("sql migrations applied (version=%d)\n", v)
	case "down":
		v, err := sqlmigrations.DownPostgres(dsn)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("sql migrations rolled back (version=%d)\n", v)
	case "version":
		v, dirty, err := sqlmigrations.VersionPostgres(dsn)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("version=%d dirty=%v expected=%d mode=%s\n", v, dirty, sqlmigrations.ExpectedVersion(), sqlmigrations.Mode())
	default:
		fmt.Fprintf(os.Stderr, "unknown schema subcommand %q\n", sub)
		os.Exit(2)
	}
}

func splitCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func mustJSON(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		log.Fatalf("marshal: %v", err)
	}
	return string(b)
}
