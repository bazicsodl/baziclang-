package intrinsics

import "strings"

func GoHelperCode() string {
	var b strings.Builder
	b.WriteString("func print(v any) { fmt.Print(v) }\n")
	b.WriteString("func println(v any) { fmt.Println(v) }\n")
	b.WriteString("func str(v any) string { return fmt.Sprint(v) }\n")
	b.WriteString("func " + LLVMRuntimeLenFunc + "(s string) int64 { return int64(utf8.RuneCountInString(s)) }\n")
	b.WriteString("func contains(s string, sub string) bool { return strings.Contains(s, sub) }\n")
	b.WriteString("func starts_with(s string, prefix string) bool { return strings.HasPrefix(s, prefix) }\n")
	b.WriteString("func ends_with(s string, suffix string) bool { return strings.HasSuffix(s, suffix) }\n")
	b.WriteString("func to_upper(s string) string { return strings.ToUpper(s) }\n")
	b.WriteString("func to_lower(s string) string { return strings.ToLower(s) }\n")
	b.WriteString("func trim_space(s string) string { return strings.TrimSpace(s) }\n")
	b.WriteString("func replace(s string, old string, new string) string { return strings.ReplaceAll(s, old, new) }\n")
	b.WriteString("func repeat(s string, count int64) string {\n")
	b.WriteString("\tif count <= 0 { return \"\" }\n")
	b.WriteString("\tmax := int64(^uint(0) >> 1)\n")
	b.WriteString("\tif count > max { count = max }\n")
	b.WriteString("\treturn strings.Repeat(s, int(count))\n")
	b.WriteString("}\n")
	b.WriteString("func parse_int(s string) Result[int64, Error] {\n")
	b.WriteString("\tv, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)\n")
	b.WriteString("\tif err != nil {\n")
	b.WriteString("\t\treturn Result[int64, Error]{Is_ok: false, Value: 0, Err: Error{Message: err.Error()}}\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn Result[int64, Error]{Is_ok: true, Value: v, Err: Error{Message: \"\"}}\n")
	b.WriteString("}\n")
	b.WriteString("func parse_float(s string) Result[float64, Error] {\n")
	b.WriteString("\tv, err := strconv.ParseFloat(strings.TrimSpace(s), 64)\n")
	b.WriteString("\tif err != nil {\n")
	b.WriteString("\t\treturn Result[float64, Error]{Is_ok: false, Value: 0.0, Err: Error{Message: err.Error()}}\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn Result[float64, Error]{Is_ok: true, Value: v, Err: Error{Message: \"\"}}\n")
	b.WriteString("}\n\n")
	return b.String()
}
