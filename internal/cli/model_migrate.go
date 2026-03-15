package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"baziclang/internal/dbmigrate"
	"baziclang/internal/modelgen"
)

func migrateCmd(binaryName string, args []string) int {
	if len(args) == 0 {
		return die(fmt.Sprintf("usage: %s migrate <create|apply|rollback|status>", binaryName))
	}
	switch args[0] {
	case "create":
		fs := flag.NewFlagSet("migrate create", flag.ExitOnError)
		dir := fs.String("dir", "migrations", "migrations directory")
		name := fs.String("name", "", "migration name")
		_ = fs.Parse(args[1:])
		if strings.TrimSpace(*name) == "" {
			return die("migration name is required")
		}
		path, err := dbmigrate.CreateMigration(*dir, *name)
		if err != nil {
			return die(err.Error())
		}
		fmt.Printf("Created %s\n", path)
		return 0
	case "apply":
		fs := flag.NewFlagSet("migrate apply", flag.ExitOnError)
		dir := fs.String("dir", "migrations", "migrations directory")
		driver := fs.String("driver", "sqlite", "db driver: sqlite|postgres|mysql")
		dsn := fs.String("dsn", "app.db", "db dsn")
		steps := fs.Int("steps", 0, "number of migrations to apply (0 = all)")
		_ = fs.Parse(args[1:])
		migrations, err := dbmigrate.LoadMigrations(*dir)
		if err != nil {
			return die(err.Error())
		}
		db, err := dbmigrate.Open(*driver, *dsn)
		if err != nil {
			return die(err.Error())
		}
		defer db.Close()
		if err := dbmigrate.ApplySteps(*driver, db, migrations, *steps); err != nil {
			return die(err.Error())
		}
		fmt.Println("Migrations applied")
		return 0
	case "rollback":
		fs := flag.NewFlagSet("migrate rollback", flag.ExitOnError)
		dir := fs.String("dir", "migrations", "migrations directory")
		driver := fs.String("driver", "sqlite", "db driver: sqlite|postgres|mysql")
		dsn := fs.String("dsn", "app.db", "db dsn")
		steps := fs.Int("steps", 1, "number of migrations to rollback")
		_ = fs.Parse(args[1:])
		if *steps < 1 {
			return die("steps must be >= 1")
		}
		migrations, err := dbmigrate.LoadMigrations(*dir)
		if err != nil {
			return die(err.Error())
		}
		db, err := dbmigrate.Open(*driver, *dsn)
		if err != nil {
			return die(err.Error())
		}
		defer db.Close()
		if err := dbmigrate.RollbackSteps(*driver, db, migrations, *steps); err != nil {
			return die(err.Error())
		}
		fmt.Println("Rollback complete")
		return 0
	case "status":
		fs := flag.NewFlagSet("migrate status", flag.ExitOnError)
		dir := fs.String("dir", "migrations", "migrations directory")
		driver := fs.String("driver", "sqlite", "db driver: sqlite|postgres|mysql")
		dsn := fs.String("dsn", "app.db", "db dsn")
		_ = fs.Parse(args[1:])
		migrations, err := dbmigrate.LoadMigrations(*dir)
		if err != nil {
			return die(err.Error())
		}
		db, err := dbmigrate.Open(*driver, *dsn)
		if err != nil {
			return die(err.Error())
		}
		defer db.Close()
		report, err := dbmigrate.StatusReport(db, migrations)
		if err != nil {
			return die(err.Error())
		}
		fmt.Print(report)
		return 0
	default:
		return die(fmt.Sprintf("unknown migrate command: %s", args[0]))
	}
}

func modelCmd(binaryName string, args []string) int {
	if len(args) == 0 {
		return die(fmt.Sprintf("usage: %s model <init|generate|migrate>", binaryName))
	}
	switch args[0] {
	case "init":
		fs := flag.NewFlagSet("model init", flag.ExitOnError)
		path := fs.String("path", "bazic.schema.json", "schema path")
		_ = fs.Parse(args[1:])
		if err := modelgen.WriteTemplate(*path); err != nil {
			return die(err.Error())
		}
		fmt.Printf("Wrote %s\n", *path)
		return 0
	case "auth":
		fs := flag.NewFlagSet("model auth", flag.ExitOnError)
		path := fs.String("path", "bazic.auth.schema.json", "schema path")
		_ = fs.Parse(args[1:])
		if err := modelgen.WriteAuthTemplate(*path); err != nil {
			return die(err.Error())
		}
		fmt.Printf("Wrote %s\n", *path)
		return 0
	case "generate":
		fs := flag.NewFlagSet("model generate", flag.ExitOnError)
		schemaPath := fs.String("schema", "bazic.schema.json", "schema path")
		out := fs.String("out", "models.bz", "output bazic file")
		_ = fs.Parse(args[1:])
		schema, err := modelgen.LoadSchema(*schemaPath)
		if err != nil {
			return die(err.Error())
		}
		if err := modelgen.ValidateSchema(schema); err != nil {
			return die(err.Error())
		}
		code := modelgen.GenerateBazic(schema)
		if err := os.WriteFile(*out, []byte(code), 0644); err != nil {
			return die(err.Error())
		}
		fmt.Printf("Wrote %s\n", *out)
		return 0
	case "migrate":
		fs := flag.NewFlagSet("model migrate", flag.ExitOnError)
		schemaPath := fs.String("schema", "bazic.schema.json", "schema path")
		migrationsDir := fs.String("migrations", "migrations", "migrations directory")
		snapshotPath := fs.String("snapshot", filepath.Join(".bazic", "schema.snapshot.json"), "snapshot path")
		name := fs.String("name", "auto", "migration name")
		driver := fs.String("driver", "", "db driver override")
		_ = fs.Parse(args[1:])
		schema, err := modelgen.LoadSchema(*schemaPath)
		if err != nil {
			return die(err.Error())
		}
		if err := modelgen.ValidateSchema(schema); err != nil {
			return die(err.Error())
		}
		useDriver := strings.TrimSpace(*driver)
		if useDriver == "" {
			useDriver = strings.TrimSpace(schema.Database.Driver)
		}
		if useDriver == "" {
			useDriver = "sqlite"
		}
		up, down, changed, err := modelgen.DetectChanges(schema, *snapshotPath, useDriver)
		if err != nil {
			return die(err.Error())
		}
		if !changed {
			fmt.Println("No schema changes detected")
			return 0
		}
		path, err := modelgen.WriteMigration(*migrationsDir, *name, up, down)
		if err != nil {
			return die(err.Error())
		}
		if err := modelgen.SaveSnapshot(schema, *snapshotPath); err != nil {
			return die(err.Error())
		}
		fmt.Printf("Wrote %s\n", path)
		return 0
	default:
		return die(fmt.Sprintf("unknown model command: %s", args[0]))
	}
}
