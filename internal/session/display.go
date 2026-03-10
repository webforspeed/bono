package session

import (
	"encoding/base64"
	"fmt"
	"strings"
)

// FormatTool returns the shared one-line tool label used by TUI and headless frontends.
func FormatTool(name string, args map[string]any) string {
	switch name {
	case "read_file":
		path, _ := args["path"].(string)
		return fmt.Sprintf("Read('%s')", path)
	case "write_file":
		path, _ := args["path"].(string)
		content, _ := args["content"].(string)
		lines := len(strings.Split(content, "\n"))
		return fmt.Sprintf("Write('%s', %d lines)", path, lines)
	case "edit_file":
		path, _ := args["path"].(string)
		return fmt.Sprintf("Edit('%s')", path)
	case "run_shell":
		cmd, _ := args["command"].(string)
		desc, _ := args["description"].(string)
		safety, _ := args["safety"].(string)
		if desc == "" {
			desc = "(no description)"
		}
		if safety == "" {
			safety = "modify"
		}
		return fmt.Sprintf("Bash('%s') # %s, %s", cmd, desc, safety)
	case "python_runtime":
		code, _ := args["code"].(string)
		desc, _ := args["description"].(string)
		safety, _ := args["safety"].(string)
		if desc == "" {
			desc = "(no description)"
		}
		if safety == "" {
			safety = "modify"
		}
		if code == "" {
			code = "(empty code)"
		}
		return fmt.Sprintf("Python(%s) # %s, %s", code, desc, safety)
	case "compact_context":
		return "Compact(context)"
	case "code_search":
		query, _ := args["query"].(string)
		searchType, _ := args["search_type"].(string)
		if searchType == "" {
			searchType = "semantic"
		}
		return fmt.Sprintf("Search('%s', %s)", query, searchType)
	case "WebSearch":
		query, _ := args["query"].(string)
		mode, _ := args["mode"].(string)
		if mode != "" {
			return fmt.Sprintf("WebSearch('%s', %s)", query, mode)
		}
		return fmt.Sprintf("WebSearch('%s')", query)
	case "WebFetch":
		url, _ := args["url"].(string)
		question, _ := args["question"].(string)
		if question != "" {
			return fmt.Sprintf("WebFetch('%s', '%s')", url, question)
		}
		return fmt.Sprintf("WebFetch('%s')", url)
	case "enter_plan_mode":
		desc, _ := args["project_description"].(string)
		if len(desc) > 50 {
			desc = desc[:50] + "..."
		}
		return fmt.Sprintf("EnterPlanMode('%s')", desc)
	default:
		return name
	}
}

func BatchReviewPrompt(count int) string {
	label := "changes"
	if count == 1 {
		label = "change"
	}
	return fmt.Sprintf("Approve %d %s or Undo", count, label)
}

func DisplaySandboxCommand(command string) string {
	if code, ok := PythonCodeFromCommand(command); ok {
		return fmt.Sprintf("Python(%s)", code)
	}
	return fmt.Sprintf("Bash(%s)", command)
}

func PythonCodeFromCommand(command string) (string, bool) {
	const marker = "base64.b64decode('"
	idx := strings.Index(command, marker)
	if idx == -1 {
		return "", false
	}
	start := idx + len(marker)
	end := strings.Index(command[start:], "')")
	if end == -1 {
		return "", false
	}
	encoded := command[start : start+end]
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", false
	}
	return string(decoded), true
}
