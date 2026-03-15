package dbmigrate

import (
	"bufio"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Migration struct {
	ID      string
	Name    string
	Path    string
	UpSQL   string
	DownSQL string
}

func EnsureMigrationsTable(db *sql.DB) error {
	_, err := db.Exec("create table if not exists bazic_migrations (id text primary key, name text not null, applied_at text not null)")
	return err
}

func LoadMigrations(dir string) ([]Migration, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	migrations := make([]Migration, 0)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}
		id := strings.TrimSuffix(name, ".sql")
		parts := strings.SplitN(id, "_", 2)
		if len(parts) < 2 {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		up, down, err := splitMigration(string(data))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		migrations = append(migrations, Migration{ID: parts[0], Name: parts[1], Path: filepath.Join(dir, name), UpSQL: up, DownSQL: down})
	}
	sort.Slice(migrations, func(i, j int) bool { return migrations[i].ID < migrations[j].ID })
	return migrations, nil
}

func CreateMigration(dir string, name string) (string, error) {
	name = sanitizeName(name)
	if name == "" {
		return "", errors.New("empty migration name")
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	id := time.Now().UTC().Format("20060102150405")
	filename := fmt.Sprintf("%s_%s.sql", id, name)
	path := filepath.Join(dir, filename)
	if _, err := os.Stat(path); err == nil {
		return "", fmt.Errorf("migration already exists: %s", path)
	}
	template := "-- +up\n\n-- +down\n"
	if err := os.WriteFile(path, []byte(template), 0644); err != nil {
		return "", err
	}
	return path, nil
}

func AppliedMigrations(db *sql.DB) (map[string]bool, error) {
	rows, err := db.Query("select id from bazic_migrations")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = true
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func ApplyAll(driver string, db *sql.DB, migrations []Migration) error {
	return ApplySteps(driver, db, migrations, 0)
}

func ApplySteps(driver string, db *sql.DB, migrations []Migration, steps int) error {
	if err := EnsureMigrationsTable(db); err != nil {
		return err
	}
	applied, err := AppliedMigrations(db)
	if err != nil {
		return err
	}
	count := 0
	for _, m := range migrations {
		if applied[m.ID] {
			continue
		}
		if steps > 0 && count >= steps {
			break
		}
		if strings.TrimSpace(m.UpSQL) == "" {
			return fmt.Errorf("migration %s has empty up section", m.Path)
		}
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		if err := execSQL(tx, m.UpSQL); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply %s: %w", m.Path, err)
		}
		if _, err := execWithDriver(driver, tx, "insert into bazic_migrations (id, name, applied_at) values (%s, %s, %s)", m.ID, m.Name, time.Now().UTC().Format(time.RFC3339)); err != nil {
			_ = tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		count++
	}
	return nil
}

func RollbackSteps(driver string, db *sql.DB, migrations []Migration, steps int) error {
	if err := EnsureMigrationsTable(db); err != nil {
		return err
	}
	appliedIDs, err := AppliedMigrations(db)
	if err != nil {
		return err
	}
	applied := []Migration{}
	for _, m := range migrations {
		if appliedIDs[m.ID] {
			applied = append(applied, m)
		}
	}
	count := 0
	for i := len(applied) - 1; i >= 0; i-- {
		if steps > 0 && count >= steps {
			break
		}
		m := applied[i]
		if strings.TrimSpace(m.DownSQL) == "" {
			return fmt.Errorf("migration %s has empty down section", m.Path)
		}
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		if err := execSQL(tx, m.DownSQL); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("rollback %s: %w", m.Path, err)
		}
		if _, err := execWithDriver(driver, tx, "delete from bazic_migrations where id = %s", m.ID); err != nil {
			_ = tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		count++
	}
	return nil
}

func StatusReport(db *sql.DB, migrations []Migration) (string, error) {
	if err := EnsureMigrationsTable(db); err != nil {
		return "", err
	}
	applied, err := AppliedMigrations(db)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for _, m := range migrations {
		mark := "down"
		if applied[m.ID] {
			mark = "up"
		}
		b.WriteString(fmt.Sprintf("%s %s_%s\n", mark, m.ID, m.Name))
	}
	return b.String(), nil
}

func splitMigration(data string) (string, string, error) {
	up := ""
	down := ""
	section := ""
	reader := bufio.NewReader(strings.NewReader(data))
	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return "", "", err
		}
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "--") {
			tag := strings.TrimSpace(strings.TrimPrefix(trimmed, "--"))
			if tag == "+up" {
				section = "up"
				if err == io.EOF {
					break
				}
				continue
			}
			if tag == "+down" {
				section = "down"
				if err == io.EOF {
					break
				}
				continue
			}
		}
		if section == "up" {
			up += line
		} else if section == "down" {
			down += line
		}
		if err == io.EOF {
			break
		}
	}
	return strings.TrimSpace(up), strings.TrimSpace(down), nil
}

type execer interface {
	Exec(query string, args ...any) (sql.Result, error)
}

func execSQL(db execer, sqlText string) error {
	stmts := splitSQLStatements(sqlText)
	for _, stmt := range stmts {
		if strings.TrimSpace(stmt) == "" {
			continue
		}
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func execWithDriver(driver string, db execer, tmpl string, args ...any) (sql.Result, error) {
	query := formatPlaceholders(driver, tmpl, len(args))
	return db.Exec(query, args...)
}

func formatPlaceholders(driver string, tmpl string, n int) string {
	driver = strings.TrimSpace(driver)
	if driver == "postgres" {
		out := tmpl
		for i := 1; i <= n; i++ {
			out = strings.Replace(out, "%s", fmt.Sprintf("$%d", i), 1)
		}
		return out
	}
	return strings.ReplaceAll(tmpl, "%s", "?")
}

func splitSQLStatements(sqlText string) []string {
	out := []string{}
	var b strings.Builder
	inSingle := false
	inDouble := false
	inBacktick := false
	inLineComment := false
	inBlockComment := false
	runes := []rune(sqlText)
	for i := 0; i < len(runes); i++ {
		c := runes[i]
		next := rune(0)
		if i+1 < len(runes) {
			next = runes[i+1]
		}
		if inLineComment {
			if c == '\n' {
				inLineComment = false
			}
			b.WriteRune(c)
			continue
		}
		if inBlockComment {
			if c == '*' && next == '/' {
				inBlockComment = false
				b.WriteRune(c)
				b.WriteRune(next)
				i++
				continue
			}
			b.WriteRune(c)
			continue
		}
		if !inSingle && !inDouble && !inBacktick {
			if c == '-' && next == '-' {
				inLineComment = true
				b.WriteRune(c)
				b.WriteRune(next)
				i++
				continue
			}
			if c == '/' && next == '*' {
				inBlockComment = true
				b.WriteRune(c)
				b.WriteRune(next)
				i++
				continue
			}
		}
		if c == '\'' && !inDouble && !inBacktick {
			inSingle = !inSingle
		} else if c == '"' && !inSingle && !inBacktick {
			inDouble = !inDouble
		} else if c == '`' && !inSingle && !inDouble {
			inBacktick = !inBacktick
		}
		if c == ';' && !inSingle && !inDouble && !inBacktick {
			out = append(out, b.String())
			b.Reset()
			continue
		}
		b.WriteRune(c)
	}
	if b.Len() > 0 {
		out = append(out, b.String())
	}
	return out
}

func sanitizeName(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, "__", "_")
	s = strings.Trim(s, "_")
	return s
}

func Open(driver string, dsn string) (*sql.DB, error) {
	driver = strings.TrimSpace(driver)
	dsn = strings.TrimSpace(dsn)
	if driver == "" {
		return nil, errors.New("empty driver")
	}
	if dsn == "" {
		return nil, errors.New("empty dsn")
	}
	switch driver {
	case "sqlite3":
		driver = "sqlite"
	case "postgresql":
		driver = "postgres"
	}
	return sql.Open(driver, dsn)
}
