package modelgen

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Schema struct {
	Version  int      `json:"version"`
	Database Database `json:"database"`
	Models   []Model  `json:"models"`
}

type Database struct {
	Driver string `json:"driver"`
	Dsn    string `json:"dsn"`
}

type Model struct {
	Name    string  `json:"name"`
	Table   string  `json:"table"`
	Fields  []Field `json:"fields"`
	Indexes []Index `json:"indexes"`
}

type Field struct {
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Optional bool     `json:"optional"`
	PK       bool     `json:"pk"`
	Auto     bool     `json:"auto"`
	Unique   bool     `json:"unique"`
	Default  string   `json:"default"`
	Ref      string   `json:"ref"`
	Size     int      `json:"size"`
	MinLen   int      `json:"min_len"`
	MaxLen   int      `json:"max_len"`
	Min      int      `json:"min"`
	Max      int      `json:"max"`
	MinF     float64  `json:"min_f"`
	MaxF     float64  `json:"max_f"`
	Enum     []string `json:"enum"`
	Join     string   `json:"join"`
}

type Index struct {
	Name   string   `json:"name"`
	Fields []string `json:"fields"`
	Unique bool     `json:"unique"`
}

func LoadSchema(path string) (Schema, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Schema{}, err
	}
	var s Schema
	if err := json.Unmarshal(data, &s); err != nil {
		return Schema{}, err
	}
	normalizeSchema(&s)
	return s, nil
}

func WriteTemplate(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("schema already exists: %s", path)
	}
	tpl := Schema{
		Version:  1,
		Database: Database{Driver: "sqlite", Dsn: "app.db"},
		Models: []Model{
			{
				Name:  "User",
				Table: "users",
				Fields: []Field{
					{Name: "id", Type: "int", PK: true, Auto: true},
					{Name: "email", Type: "string", Unique: true},
					{Name: "name", Type: "string", Optional: true},
					{Name: "created_at", Type: "time", Default: "raw:CURRENT_TIMESTAMP"},
				},
				Indexes: []Index{
					{Name: "idx_users_email", Fields: []string{"email"}, Unique: true},
				},
			},
		},
	}
	data, _ := json.MarshalIndent(tpl, "", "  ")
	return os.WriteFile(path, data, 0644)
}

func WriteAuthTemplate(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("schema already exists: %s", path)
	}
	tpl := Schema{
		Version:  1,
		Database: Database{Driver: "sqlite", Dsn: "app.db"},
		Models: []Model{
			{
				Name:  "User",
				Table: "users",
				Fields: []Field{
					{Name: "id", Type: "int", PK: true, Auto: true},
					{Name: "email", Type: "string", Unique: true},
					{Name: "password_hash", Type: "string"},
					{Name: "name", Type: "string", Optional: true},
					{Name: "created_at", Type: "time", Default: "raw:CURRENT_TIMESTAMP"},
				},
				Indexes: []Index{
					{Name: "idx_users_email", Fields: []string{"email"}, Unique: true},
				},
			},
			{
				Name:  "Session",
				Table: "sessions",
				Fields: []Field{
					{Name: "id", Type: "int", PK: true, Auto: true},
					{Name: "token_hash", Type: "string", Unique: true},
					{Name: "user_id", Type: "int"},
					{Name: "expires_at", Type: "time"},
					{Name: "created_at", Type: "time", Default: "raw:CURRENT_TIMESTAMP"},
				},
				Indexes: []Index{
					{Name: "idx_sessions_user", Fields: []string{"user_id"}, Unique: false},
				},
			},
		},
	}
	data, _ := json.MarshalIndent(tpl, "", "  ")
	return os.WriteFile(path, data, 0644)
}

func GenerateBazic(schema Schema) string {
	var b strings.Builder
	b.WriteString("import \"std\";\n\n")
	b.WriteString("fn sql_escape(s: string): string { return replace(s, \"'\", \"''\"); }\n")
	b.WriteString("fn sql_str(s: string): string { return \"'\" + sql_escape(s) + \"'\"; }\n")
	b.WriteString("fn sql_bool(b: bool): string { if b { return \"1\"; } return \"0\"; }\n")
	b.WriteString("fn sql_opt_str(v: Option[string]): string { if v.is_some { return sql_str(v.value); } return \"null\"; }\n")
	b.WriteString("fn sql_opt_int(v: Option[int]): string { if v.is_some { return str(v.value); } return \"null\"; }\n")
	b.WriteString("fn sql_opt_float(v: Option[float]): string { if v.is_some { return str(v.value); } return \"null\"; }\n")
	b.WriteString("fn sql_opt_bool(v: Option[bool]): string { if v.is_some { return sql_bool(v.value); } return \"null\"; }\n\n")
	b.WriteString("fn sql_order_by(field: string, asc: bool): string { if field == \"\" { return \"\"; } if asc { return \" order by \" + field + \" asc\"; } return \" order by \" + field + \" desc\"; }\n")
	b.WriteString("fn sql_limit_offset(limit: int, offset: int): string { let s = \"\"; if limit > 0 { s = s + \" limit \" + str(limit); } if offset > 0 { s = s + \" offset \" + str(offset); } return s; }\n\n")
	for _, m := range schema.Models {
		b.WriteString("struct ")
		b.WriteString(m.Name)
		b.WriteString(" {\n")
		for _, f := range m.Fields {
			b.WriteString("    ")
			b.WriteString(f.Name)
			b.WriteString(": ")
			b.WriteString(mapBazicType(f.Type, f.Optional))
			b.WriteString(";\n")
		}
		b.WriteString("}\n\n")
		lower := strings.ToLower(m.Name)
		b.WriteString("fn ")
		b.WriteString(lower)
		b.WriteString("_table(): string { return \"")
		b.WriteString(m.Table)
		b.WriteString("\"; }\n")
		b.WriteString("fn ")
		b.WriteString(lower)
		b.WriteString("_columns(): string { return \"")
		b.WriteString(strings.Join(fieldNames(m.Fields), ","))
		b.WriteString("\"; }\n\n")
		b.WriteString("fn ")
		b.WriteString(lower)
		b.WriteString("_zero(): ")
		b.WriteString(m.Name)
		b.WriteString(" {\n")
		b.WriteString("    return ")
		b.WriteString(m.Name)
		b.WriteString("{\n")
		for _, f := range m.Fields {
			b.WriteString("        ")
			b.WriteString(f.Name)
			b.WriteString(": ")
			b.WriteString(zeroValue(f))
			b.WriteString(",\n")
		}
		b.WriteString("    };\n")
		b.WriteString("}\n\n")

		b.WriteString("fn ")
		b.WriteString(lower)
		b.WriteString("_from_json(s: string): Result[")
		b.WriteString(m.Name)
		b.WriteString(",Error] {\n")
		b.WriteString("    let zero = ")
		b.WriteString(lower)
		b.WriteString("_zero();\n")
		for _, f := range m.Fields {
			parse := jsonGetter(f)
			if f.Optional {
				b.WriteString("    let ")
				b.WriteString(f.Name)
				b.WriteString("_opt = none(")
				b.WriteString(optionFallback(f))
				b.WriteString(");\n")
				b.WriteString("    let ")
				b.WriteString(f.Name)
				b.WriteString("_raw = json_get_raw(s, \"")
				b.WriteString(f.Name)
				b.WriteString("\");\n")
				b.WriteString("    if ")
				b.WriteString(f.Name)
				b.WriteString("_raw.is_ok {\n")
				b.WriteString("        if ")
				b.WriteString(f.Name)
				b.WriteString("_raw.value == \"null\" {\n")
				b.WriteString("        } else {\n")
				b.WriteString("        let ")
				b.WriteString(f.Name)
				b.WriteString("_res = ")
				b.WriteString(parse)
				b.WriteString(";\n")
				b.WriteString("        if !")
				b.WriteString(f.Name)
				b.WriteString("_res.is_ok { return err(zero, ")
				b.WriteString(f.Name)
				b.WriteString("_res.err); }\n")
				b.WriteString("        ")
				b.WriteString(f.Name)
				b.WriteString("_opt = some(")
				b.WriteString(f.Name)
				b.WriteString("_res.value);\n")
				b.WriteString("        }\n")
				b.WriteString("    }\n")
			} else {
				b.WriteString("    let ")
				b.WriteString(f.Name)
				b.WriteString("_res = ")
				b.WriteString(parse)
				b.WriteString(";\n")
				b.WriteString("    if !")
				b.WriteString(f.Name)
				b.WriteString("_res.is_ok { return err(zero, ")
				b.WriteString(f.Name)
				b.WriteString("_res.err); }\n")
			}
		}
		b.WriteString("    return ok(")
		b.WriteString(m.Name)
		b.WriteString("{\n")
		for _, f := range m.Fields {
			b.WriteString("        ")
			b.WriteString(f.Name)
			b.WriteString(": ")
			if f.Optional {
				b.WriteString(f.Name)
				b.WriteString("_opt")
			} else {
				b.WriteString(f.Name)
				b.WriteString("_res.value")
			}
			b.WriteString(",\n")
		}
		b.WriteString("    }, Error{message: \"\"});\n")
		b.WriteString("}\n\n")

		b.WriteString("fn ")
		b.WriteString(lower)
		b.WriteString("_validate(item: ")
		b.WriteString(m.Name)
		b.WriteString("): Result[bool,Error] {\n")
		for _, f := range m.Fields {
			if f.Optional {
				continue
			}
			if strings.ToLower(strings.TrimSpace(f.Type)) == "string" || strings.ToLower(strings.TrimSpace(f.Type)) == "text" {
				b.WriteString("    let ")
				b.WriteString(f.Name)
				b.WriteString("_req = validate_required(item.")
				b.WriteString(f.Name)
				b.WriteString(", \"")
				b.WriteString(f.Name)
				b.WriteString("\");\n")
				b.WriteString("    if !")
				b.WriteString(f.Name)
				b.WriteString("_req.is_ok { return err(false, ")
				b.WriteString(f.Name)
				b.WriteString("_req.err); }\n")
				if f.MinLen > 0 {
					b.WriteString("    let ")
					b.WriteString(f.Name)
					b.WriteString("_min = validate_min_len(item.")
					b.WriteString(f.Name)
					b.WriteString(", ")
					b.WriteString(strconv.Itoa(f.MinLen))
					b.WriteString(", \"")
					b.WriteString(f.Name)
					b.WriteString("\");\n")
					b.WriteString("    if !")
					b.WriteString(f.Name)
					b.WriteString("_min.is_ok { return err(false, ")
					b.WriteString(f.Name)
					b.WriteString("_min.err); }\n")
				}
				if f.MaxLen > 0 {
					b.WriteString("    let ")
					b.WriteString(f.Name)
					b.WriteString("_max = validate_max_len(item.")
					b.WriteString(f.Name)
					b.WriteString(", ")
					b.WriteString(strconv.Itoa(f.MaxLen))
					b.WriteString(", \"")
					b.WriteString(f.Name)
					b.WriteString("\");\n")
					b.WriteString("    if !")
					b.WriteString(f.Name)
					b.WriteString("_max.is_ok { return err(false, ")
					b.WriteString(f.Name)
					b.WriteString("_max.err); }\n")
				}
				if len(f.Enum) > 0 {
					b.WriteString("    let ")
					b.WriteString(f.Name)
					b.WriteString("_enum = validate_enum(item.")
					b.WriteString(f.Name)
					b.WriteString(", \"")
					b.WriteString(strings.Join(f.Enum, ","))
					b.WriteString("\");\n")
					b.WriteString("    if !")
					b.WriteString(f.Name)
					b.WriteString("_enum.is_ok { return err(false, ")
					b.WriteString(f.Name)
					b.WriteString("_enum.err); }\n")
				}
				if f.Unique || strings.Contains(strings.ToLower(f.Name), "email") {
					b.WriteString("    let ")
					b.WriteString(f.Name)
					b.WriteString("_email = validate_email(item.")
					b.WriteString(f.Name)
					b.WriteString(", \"")
					b.WriteString(f.Name)
					b.WriteString("\");\n")
					b.WriteString("    if !")
					b.WriteString(f.Name)
					b.WriteString("_email.is_ok { return err(false, ")
					b.WriteString(f.Name)
					b.WriteString("_email.err); }\n")
				}
			} else if strings.ToLower(strings.TrimSpace(f.Type)) == "int" {
				if f.Min != 0 || f.Max != 0 {
					b.WriteString("    let ")
					b.WriteString(f.Name)
					b.WriteString("_range = validate_int_range(item.")
					b.WriteString(f.Name)
					b.WriteString(", ")
					b.WriteString(strconv.Itoa(f.Min))
					b.WriteString(", ")
					b.WriteString(strconv.Itoa(f.Max))
					b.WriteString(", \"")
					b.WriteString(f.Name)
					b.WriteString("\");\n")
					b.WriteString("    if !")
					b.WriteString(f.Name)
					b.WriteString("_range.is_ok { return err(false, ")
					b.WriteString(f.Name)
					b.WriteString("_range.err); }\n")
				}
			} else if strings.ToLower(strings.TrimSpace(f.Type)) == "float" {
				if f.MinF != 0 || f.MaxF != 0 {
					b.WriteString("    let ")
					b.WriteString(f.Name)
					b.WriteString("_range = validate_float_range(item.")
					b.WriteString(f.Name)
					b.WriteString(", ")
					b.WriteString(fmt.Sprintf("%g", f.MinF))
					b.WriteString(", ")
					b.WriteString(fmt.Sprintf("%g", f.MaxF))
					b.WriteString(", \"")
					b.WriteString(f.Name)
					b.WriteString("\");\n")
					b.WriteString("    if !")
					b.WriteString(f.Name)
					b.WriteString("_range.is_ok { return err(false, ")
					b.WriteString(f.Name)
					b.WriteString("_range.err); }\n")
				}
			}
		}
		b.WriteString("    return ok(true, Error{message: \"\"});\n")
		b.WriteString("}\n\n")

		pk := primaryKeyField(m.Fields)
		if pk != nil {
			b.WriteString("fn ")
			b.WriteString(lower)
			b.WriteString("_find_by_id(db_path: string, id: ")
			b.WriteString(mapBazicType(pk.Type, false))
			b.WriteString("): Result[")
			b.WriteString(m.Name)
			b.WriteString(",Error] {\n")
			b.WriteString("    let res = db_query_one_json(db_path, \"select \" + ")
			b.WriteString(lower)
			b.WriteString("_columns() + \" from ")
			b.WriteString(m.Table)
			b.WriteString(" where ")
			b.WriteString(pk.Name)
			b.WriteString(" = \" + str(id));\n")
			b.WriteString("    if !res.is_ok { return err(")
			b.WriteString(lower)
			b.WriteString("_zero(), res.err); }\n")
			b.WriteString("    return ")
			b.WriteString(lower)
			b.WriteString("_from_json(res.value);\n")
			b.WriteString("}\n\n")
		}

		for _, idx := range m.Indexes {
			if len(idx.Fields) == 1 {
				field := idx.Fields[0]
				f, ok := fieldByName(m.Fields, field)
				if !ok {
					continue
				}
				b.WriteString("fn ")
				b.WriteString(lower)
				b.WriteString("_find_by_")
				b.WriteString(field)
				b.WriteString("(db_path: string, v: ")
				b.WriteString(mapBazicType(f.Type, false))
				b.WriteString("): Result[")
				b.WriteString(m.Name)
				b.WriteString(",Error] {\n")
				b.WriteString("    let sql = \"select \" + ")
				b.WriteString(lower)
				b.WriteString("_columns() + \" from ")
				b.WriteString(m.Table)
				b.WriteString(" where ")
				b.WriteString(field)
				b.WriteString(" = \" + ")
				b.WriteString(sqlValueExpr("v", *f))
				b.WriteString(";\n")
				b.WriteString("    let res = db_query_one_json(db_path, sql);\n")
				b.WriteString("    if !res.is_ok { return err(")
				b.WriteString(lower)
				b.WriteString("_zero(), res.err); }\n")
				b.WriteString("    return ")
				b.WriteString(lower)
				b.WriteString("_from_json(res.value);\n")
				b.WriteString("}\n\n")
			}
		}

		for _, f := range m.Fields {
			if f.Optional {
				continue
			}
			b.WriteString("fn ")
			b.WriteString(lower)
			b.WriteString("_list_by_")
			b.WriteString(f.Name)
			b.WriteString("_json(db_path: string, v: ")
			b.WriteString(mapBazicType(f.Type, false))
			b.WriteString("): Result[string,Error] {\n")
			b.WriteString("    let sql = \"select \" + ")
			b.WriteString(lower)
			b.WriteString("_columns() + \" from ")
			b.WriteString(m.Table)
			b.WriteString(" where ")
			b.WriteString(f.Name)
			b.WriteString(" = \" + ")
			b.WriteString(sqlValueExpr("v", f))
			b.WriteString(";\n")
			b.WriteString("    return db_query_json(db_path, sql);\n")
			b.WriteString("}\n\n")
		}

		for _, f := range m.Fields {
			if f.Ref == "" {
				continue
			}
			refParts := strings.Split(f.Ref, ".")
			if len(refParts) != 2 {
				continue
			}
			refModel := refParts[0]
			refField := refParts[1]
			b.WriteString("fn ")
			b.WriteString(lower)
			b.WriteString("_list_by_")
			b.WriteString(f.Name)
			b.WriteString("_json(db_path: string, v: ")
			b.WriteString(mapBazicType(f.Type, false))
			b.WriteString("): Result[string,Error] {\n")
			b.WriteString("    let sql = \"select \" + ")
			b.WriteString(lower)
			b.WriteString("_columns() + \" from ")
			b.WriteString(m.Table)
			b.WriteString(" where ")
			b.WriteString(f.Name)
			b.WriteString(" = \" + ")
			b.WriteString(sqlValueExpr("v", f))
			b.WriteString(";\n")
			b.WriteString("    return db_query_json(db_path, sql);\n")
			b.WriteString("}\n\n")

			b.WriteString("fn ")
			b.WriteString(lower)
			b.WriteString("_list_with_")
			b.WriteString(strings.ToLower(refModel))
			b.WriteString("_json(db_path: string): Result[string,Error] {\n")
			b.WriteString("    let sql = \"select ")
			b.WriteString(m.Table)
			b.WriteString(".* , ")
			b.WriteString(strings.ToLower(refModel))
			b.WriteString("s.* from ")
			b.WriteString(m.Table)
			b.WriteString(" join ")
			b.WriteString(strings.ToLower(refModel))
			b.WriteString("s on ")
			b.WriteString(m.Table)
			b.WriteString(".")
			b.WriteString(f.Name)
			b.WriteString(" = ")
			b.WriteString(strings.ToLower(refModel))
			b.WriteString("s.")
			b.WriteString(refField)
			b.WriteString("\";\n")
			b.WriteString("    return db_query_json(db_path, sql);\n")
			b.WriteString("}\n\n")
		}

		for _, f := range m.Fields {
			if f.Join == "" {
				continue
			}
			parts := strings.Split(f.Join, ".")
			if len(parts) != 2 {
				continue
			}
			joinModel := parts[0]
			joinField := parts[1]
			b.WriteString("fn ")
			b.WriteString(lower)
			b.WriteString("_list_with_")
			b.WriteString(strings.ToLower(joinModel))
			b.WriteString("_nested_json(db_path: string): Result[string,Error] {\n")
			b.WriteString("    let sql = \"select ")
			b.WriteString(m.Table)
			b.WriteString(".* , ")
			b.WriteString(strings.ToLower(joinModel))
			b.WriteString("s.")
			b.WriteString(joinField)
			b.WriteString(" as _join_")
			b.WriteString(strings.ToLower(joinModel))
			b.WriteString("_")
			b.WriteString(joinField)
			b.WriteString(" from ")
			b.WriteString(m.Table)
			b.WriteString(" join ")
			b.WriteString(strings.ToLower(joinModel))
			b.WriteString("s on ")
			b.WriteString(m.Table)
			b.WriteString(".")
			b.WriteString(f.Name)
			b.WriteString(" = ")
			b.WriteString(strings.ToLower(joinModel))
			b.WriteString("s.")
			b.WriteString(joinField)
			b.WriteString("\";\n")
			b.WriteString("    let res = db_query_json(db_path, sql);\n")
			b.WriteString("    if !res.is_ok { return res; }\n")
			b.WriteString("    return res;\n")
			b.WriteString("}\n\n")
		}

		b.WriteString("fn ")
		b.WriteString(lower)
		b.WriteString("_list_json(db_path: string): Result[string,Error] {\n")
		b.WriteString("    return db_query_json(db_path, \"select \" + ")
		b.WriteString(lower)
		b.WriteString("_columns() + \" from ")
		b.WriteString(m.Table)
		b.WriteString("\");\n")
		b.WriteString("}\n\n")

		b.WriteString("fn ")
		b.WriteString(lower)
		b.WriteString("_list_paged_json(db_path: string, limit: int, offset: int, order_by: string, asc: bool): Result[string,Error] {\n")
		b.WriteString("    let sql = \"select \" + ")
		b.WriteString(lower)
		b.WriteString("_columns() + \" from ")
		b.WriteString(m.Table)
		b.WriteString("\" + sql_order_by(order_by, asc) + sql_limit_offset(limit, offset);\n")
		b.WriteString("    return db_query_json(db_path, sql);\n")
		b.WriteString("}\n\n")

		insertFields := insertableFields(m.Fields)
		if len(insertFields) > 0 {
			b.WriteString("fn ")
			b.WriteString(lower)
			b.WriteString("_create(db_path: string, item: ")
			b.WriteString(m.Name)
			b.WriteString("): Result[int,Error] {\n")
			b.WriteString("    let v = ")
			b.WriteString(lower)
			b.WriteString("_validate(item);\n")
			b.WriteString("    if !v.is_ok { return err(0, v.err); }\n")
			b.WriteString("    let sql = \"insert into ")
			b.WriteString(m.Table)
			b.WriteString(" (")
			b.WriteString(strings.Join(fieldNames(insertFields), ","))
			b.WriteString(") values (\" + ")
			b.WriteString(insertValueExpr(insertFields))
			b.WriteString(" + \")\";\n")
			b.WriteString("    return db_exec_returning_id(db_path, sql);\n")
			b.WriteString("}\n\n")
		}

		if pk != nil {
			b.WriteString("fn ")
			b.WriteString(lower)
			b.WriteString("_update(db_path: string, item: ")
			b.WriteString(m.Name)
			b.WriteString("): Result[bool,Error] {\n")
			b.WriteString("    let sql = \"update ")
			b.WriteString(m.Table)
			b.WriteString(" set ")
			b.WriteString(updateSetExpr(m.Fields, pk.Name))
			b.WriteString(" where ")
			b.WriteString(pk.Name)
			b.WriteString(" = \" + str(item.")
			b.WriteString(pk.Name)
			b.WriteString(");\n")
			b.WriteString("    return db_exec(db_path, sql);\n")
			b.WriteString("}\n\n")

			b.WriteString("fn ")
			b.WriteString(lower)
			b.WriteString("_delete(db_path: string, id: ")
			b.WriteString(mapBazicType(pk.Type, false))
			b.WriteString("): Result[bool,Error] {\n")
			b.WriteString("    return db_exec(db_path, \"delete from ")
			b.WriteString(m.Table)
			b.WriteString(" where ")
			b.WriteString(pk.Name)
			b.WriteString(" = \" + str(id));\n")
			b.WriteString("}\n\n")
		}
	}
	return b.String()
}

func primaryKeyField(fields []Field) *Field {
	for _, f := range fields {
		if f.PK {
			return &f
		}
	}
	return nil
}

func insertableFields(fields []Field) []Field {
	out := []Field{}
	for _, f := range fields {
		if f.Auto && f.PK {
			continue
		}
		out = append(out, f)
	}
	return out
}

func zeroValue(f Field) string {
	if f.Optional {
		return "none(" + optionFallback(f) + ")"
	}
	switch strings.ToLower(strings.TrimSpace(f.Type)) {
	case "int":
		return "0"
	case "float":
		return "0.0"
	case "bool":
		return "false"
	default:
		return "\"\""
	}
}

func optionFallback(f Field) string {
	switch strings.ToLower(strings.TrimSpace(f.Type)) {
	case "int":
		return "0"
	case "float":
		return "0.0"
	case "bool":
		return "false"
	default:
		return "\"\""
	}
}

func jsonGetter(f Field) string {
	key := f.Name
	switch strings.ToLower(strings.TrimSpace(f.Type)) {
	case "int":
		return "json_get_int(s, \"" + key + "\")"
	case "float":
		return "json_get_float(s, \"" + key + "\")"
	case "bool":
		return "json_get_bool(s, \"" + key + "\")"
	default:
		return "json_get_string(s, \"" + key + "\")"
	}
}

func fieldByName(fields []Field, name string) (*Field, bool) {
	for _, f := range fields {
		if f.Name == name {
			return &f, true
		}
	}
	return nil, false
}

func insertValueExpr(fields []Field) string {
	parts := []string{}
	for _, f := range fields {
		expr := ""
		switch strings.ToLower(strings.TrimSpace(f.Type)) {
		case "int":
			if f.Optional {
				expr = "sql_opt_int(item." + f.Name + ")"
			} else {
				expr = "str(item." + f.Name + ")"
			}
		case "float":
			if f.Optional {
				expr = "sql_opt_float(item." + f.Name + ")"
			} else {
				expr = "str(item." + f.Name + ")"
			}
		case "bool":
			if f.Optional {
				expr = "sql_opt_bool(item." + f.Name + ")"
			} else {
				expr = "sql_bool(item." + f.Name + ")"
			}
		default:
			if f.Optional {
				expr = "sql_opt_str(item." + f.Name + ")"
			} else {
				expr = "sql_str(item." + f.Name + ")"
			}
		}
		parts = append(parts, expr)
	}
	return strings.Join(parts, " + \",\" + ")
}

func updateSetExpr(fields []Field, pkName string) string {
	parts := []string{}
	for _, f := range fields {
		if f.Name == pkName {
			continue
		}
		val := ""
		switch strings.ToLower(strings.TrimSpace(f.Type)) {
		case "int":
			if f.Optional {
				val = "sql_opt_int(item." + f.Name + ")"
			} else {
				val = "str(item." + f.Name + ")"
			}
		case "float":
			if f.Optional {
				val = "sql_opt_float(item." + f.Name + ")"
			} else {
				val = "str(item." + f.Name + ")"
			}
		case "bool":
			if f.Optional {
				val = "sql_opt_bool(item." + f.Name + ")"
			} else {
				val = "sql_bool(item." + f.Name + ")"
			}
		default:
			if f.Optional {
				val = "sql_opt_str(item." + f.Name + ")"
			} else {
				val = "sql_str(item." + f.Name + ")"
			}
		}
		parts = append(parts, "\""+f.Name+" = \" + "+val)
	}
	return strings.Join(parts, " + \",\" + ")
}

func sqlValueExpr(name string, f Field) string {
	switch strings.ToLower(strings.TrimSpace(f.Type)) {
	case "int":
		return "str(" + name + ")"
	case "float":
		return "str(" + name + ")"
	case "bool":
		return "sql_bool(" + name + ")"
	default:
		return "sql_str(" + name + ")"
	}
}

func GenerateMigration(schema Schema, prev *Schema, driver string) (string, string, bool, error) {
	changes := migrationChanges(schema, prev)
	if len(changes.Up) == 0 {
		return "", "", false, nil
	}
	up := strings.Join(changes.Up, "\n")
	down := strings.Join(changes.Down, "\n")
	if strings.TrimSpace(down) == "" {
		down = "select 1;"
	}
	return up, down, true, nil
}

func WriteMigration(dir string, name string, up string, down string) (string, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	id := time.Now().UTC().Format("20060102150405")
	filename := fmt.Sprintf("%s_%s.sql", id, sanitizeName(name))
	path := filepath.Join(dir, filename)
	data := fmt.Sprintf("-- +up\n%s\n\n-- +down\n%s\n", up, down)
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		return "", err
	}
	return path, nil
}

func SaveSnapshot(schema Schema, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, _ := json.MarshalIndent(schema, "", "  ")
	return os.WriteFile(path, data, 0644)
}

func migrationChanges(schema Schema, prev *Schema) changeSet {
	if prev == nil {
		up := []string{}
		down := []string{}
		for _, m := range schema.Models {
			up = append(up, createTableSQL(m, schema.Database.Driver))
			for _, idx := range indexSQL(m) {
				up = append(up, idx)
			}
			down = append(down, dropTableSQL(m))
		}
		return changeSet{Up: up, Down: down}
	}
	prevModels := map[string]Model{}
	for _, m := range prev.Models {
		prevModels[m.Name] = m
	}
	up := []string{}
	down := []string{}
	for _, m := range schema.Models {
		pm, ok := prevModels[m.Name]
		if !ok {
			up = append(up, createTableSQL(m, schema.Database.Driver))
			for _, idx := range indexSQL(m) {
				up = append(up, idx)
			}
			down = append(down, dropTableSQL(m))
			continue
		}
		prevFields := map[string]Field{}
		for _, f := range pm.Fields {
			prevFields[f.Name] = f
		}
		for _, f := range m.Fields {
			if _, ok := prevFields[f.Name]; ok {
				continue
			}
			up = append(up, alterAddColumnSQL(m, f, schema.Database.Driver))
			if f.Unique {
				up = append(up, uniqueIndexSQL(m, f))
			}
		}
		prevIdx := map[string]Index{}
		for _, idx := range pm.Indexes {
			prevIdx[idx.Name] = idx
		}
		for _, idx := range m.Indexes {
			if _, ok := prevIdx[idx.Name]; ok {
				continue
			}
			up = append(up, createIndexSQL(m, idx))
		}
	}
	return changeSet{Up: up, Down: down}
}

type changeSet struct {
	Up   []string
	Down []string
}

func createTableSQL(m Model, driver string) string {
	cols := []string{}
	pkCols := []string{}
	fks := []string{}
	for _, f := range m.Fields {
		col := columnSQL(f, driver)
		cols = append(cols, col)
		if f.PK && !(f.Auto && driver == "sqlite") {
			pkCols = append(pkCols, f.Name)
		}
		if f.Ref != "" {
			parts := strings.Split(f.Ref, ".")
			if len(parts) == 2 {
				refTable := strings.ToLower(parts[0]) + "s"
				fks = append(fks, fmt.Sprintf("foreign key (%s) references %s(%s) on delete cascade", f.Name, refTable, parts[1]))
			}
		}
	}
	if len(pkCols) > 0 {
		cols = append(cols, fmt.Sprintf("primary key (%s)", strings.Join(pkCols, ",")))
	}
	cols = append(cols, fks...)
	return fmt.Sprintf("create table if not exists %s (\n  %s\n);", m.Table, strings.Join(cols, ",\n  "))
}

func dropTableSQL(m Model) string {
	return fmt.Sprintf("drop table if exists %s;", m.Table)
}

func alterAddColumnSQL(m Model, f Field, driver string) string {
	return fmt.Sprintf("alter table %s add column %s;", m.Table, columnSQL(f, driver))
}

func indexSQL(m Model) []string {
	out := []string{}
	for _, idx := range m.Indexes {
		out = append(out, createIndexSQL(m, idx))
	}
	return out
}

func createIndexSQL(m Model, idx Index) string {
	name := idx.Name
	if name == "" {
		name = fmt.Sprintf("idx_%s_%s", m.Table, strings.Join(idx.Fields, "_"))
	}
	unique := ""
	if idx.Unique {
		unique = "unique "
	}
	return fmt.Sprintf("create %sindex if not exists %s on %s (%s);", unique, name, m.Table, strings.Join(idx.Fields, ","))
}

func uniqueIndexSQL(m Model, f Field) string {
	name := fmt.Sprintf("ux_%s_%s", m.Table, f.Name)
	return fmt.Sprintf("create unique index if not exists %s on %s (%s);", name, m.Table, f.Name)
}

func columnSQL(f Field, driver string) string {
	if f.PK && f.Auto {
		switch driver {
		case "postgres":
			return fmt.Sprintf("%s bigserial primary key", f.Name)
		case "mysql":
			return fmt.Sprintf("%s bigint auto_increment primary key", f.Name)
		default:
			return fmt.Sprintf("%s integer primary key autoincrement", f.Name)
		}
	}
	typ := mapSQLType(f, driver)
	parts := []string{f.Name, typ}
	if !f.Optional {
		parts = append(parts, "not null")
	}
	if f.Unique {
		parts = append(parts, "unique")
	}
	if f.Default != "" {
		parts = append(parts, "default "+formatDefault(f.Default))
	}
	return strings.Join(parts, " ")
}

func mapSQLType(f Field, driver string) string {
	t := strings.ToLower(strings.TrimSpace(f.Type))
	switch t {
	case "int":
		if driver == "postgres" {
			return "bigint"
		}
		if driver == "mysql" {
			return "bigint"
		}
		return "integer"
	case "float":
		if driver == "postgres" {
			return "double precision"
		}
		return "double"
	case "bool":
		if driver == "mysql" {
			return "tinyint(1)"
		}
		return "boolean"
	case "time":
		return "timestamp"
	case "json":
		if driver == "sqlite" {
			return "text"
		}
		return "json"
	case "text":
		return "text"
	case "string":
		if f.Size > 0 {
			return fmt.Sprintf("varchar(%d)", f.Size)
		}
		return "text"
	default:
		return "text"
	}
}

func formatDefault(v string) string {
	v = strings.TrimSpace(v)
	if strings.HasPrefix(v, "raw:") {
		return strings.TrimPrefix(v, "raw:")
	}
	return "'" + strings.ReplaceAll(v, "'", "''") + "'"
}

func mapBazicType(t string, optional bool) string {
	bt := "string"
	switch strings.ToLower(strings.TrimSpace(t)) {
	case "int":
		bt = "int"
	case "float":
		bt = "float"
	case "bool":
		bt = "bool"
	case "time":
		bt = "string"
	case "json":
		bt = "string"
	case "text":
		bt = "string"
	case "string":
		bt = "string"
	default:
		bt = "string"
	}
	if optional {
		return fmt.Sprintf("Option[%s]", bt)
	}
	return bt
}

func fieldNames(fields []Field) []string {
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		out = append(out, f.Name)
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

func normalizeSchema(s *Schema) {
	s.Database.Driver = strings.TrimSpace(s.Database.Driver)
	s.Database.Dsn = strings.TrimSpace(s.Database.Dsn)
	for mi := range s.Models {
		if s.Models[mi].Table == "" {
			s.Models[mi].Table = strings.ToLower(s.Models[mi].Name) + "s"
		}
		for fi := range s.Models[mi].Fields {
			s.Models[mi].Fields[fi].Type = strings.TrimSpace(s.Models[mi].Fields[fi].Type)
		}
		sort.Slice(s.Models[mi].Fields, func(i, j int) bool { return s.Models[mi].Fields[i].Name < s.Models[mi].Fields[j].Name })
	}
	sort.Slice(s.Models, func(i, j int) bool { return s.Models[i].Name < s.Models[j].Name })
}

func LoadSnapshot(path string) (*Schema, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s Schema
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	normalizeSchema(&s)
	return &s, nil
}

func DetectChanges(schema Schema, snapshotPath string, driver string) (string, string, bool, error) {
	var prev *Schema
	if _, err := os.Stat(snapshotPath); err == nil {
		snap, err := LoadSnapshot(snapshotPath)
		if err != nil {
			return "", "", false, err
		}
		prev = snap
	}
	up, down, changed, err := GenerateMigration(schema, prev, driver)
	if err != nil {
		return "", "", false, err
	}
	return up, down, changed, nil
}

func ValidateSchema(schema Schema) error {
	if schema.Version == 0 {
		return errors.New("schema.version is required")
	}
	if len(schema.Models) == 0 {
		return errors.New("schema.models is empty")
	}
	modelNames := map[string]bool{}
	for _, m := range schema.Models {
		if m.Name == "" {
			return errors.New("model name required")
		}
		if modelNames[m.Name] {
			return fmt.Errorf("duplicate model: %s", m.Name)
		}
		modelNames[m.Name] = true
		fieldNames := map[string]bool{}
		for _, f := range m.Fields {
			if f.Name == "" {
				return fmt.Errorf("model %s has field with empty name", m.Name)
			}
			if fieldNames[f.Name] {
				return fmt.Errorf("model %s duplicate field %s", m.Name, f.Name)
			}
			fieldNames[f.Name] = true
		}
	}
	return nil
}
