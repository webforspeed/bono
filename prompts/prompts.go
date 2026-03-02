package prompts

import (
	"bytes"
	"embed"
	"fmt"
	"strings"
	"text/template"
)

//go:embed versions/*.tmpl
var versionFS embed.FS

// HostContext holds workspace and host information to inject into the system prompt.
type HostContext struct {
	CWD      string // current working directory
	OS       string // e.g. darwin, linux
	Arch     string // e.g. arm64, amd64
	Username string
	DateTime string // current date and time
}

// AgentIdentity describes what the agent is and where it runs.
// Swappable: "coding" + "terminal" is one combination, but callers can set
// "research" + "web" or any other pairing.
type AgentIdentity struct {
	Role     string // e.g. "coding", "research"
	Platform string // e.g. "terminal", "web", "ide"
}

// PromptContext is the full data passed to system prompt templates.
// HostContext is embedded so existing {{.CWD}}, {{.OS}}, etc. keep working.
type PromptContext struct {
	HostContext
	Identity AgentIdentity
}

const defaultSystemPromptTemplate = `You are Bono, a helpful coding assistant running in a terminal.
You are operating in the user's workspace.

Host context:
- Current working directory: {{.CWD}}
- OS: {{.OS}} ({{.Arch}})
- User: {{.Username}}
- Current date/time: {{.DateTime}}

You have access to several tools to help you with your tasks.
Project instructions are in the AGENTS.md or CLAUDE.md file.
You are operating with a limited context window.
You can see how much context you've used in the tool results. Summarize findings as you go rather than accumulating raw content and risk context being full
Help the user achieve their goals.`

func renderSystemTemplate(tmpl string, ctx PromptContext) (string, error) {
	parsed, err := template.New("system_prompt").Option("missingkey=error").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var out bytes.Buffer
	if err := parsed.Execute(&out, ctx); err != nil {
		return "", err
	}

	return out.String(), nil
}

// SystemWithContext returns the system prompt with the given context injected.
func SystemWithContext(ctx PromptContext) string {
	rendered, err := renderSystemTemplate(defaultSystemPromptTemplate, ctx)
	if err != nil {
		return fmt.Sprintf("You are Bono, a helpful %s assistant running in a %s.\nCurrent working directory: %s", ctx.Identity.Role, ctx.Identity.Platform, ctx.CWD)
	}
	return rendered
}

// LoadSystemPromptVersion loads prompts/versions/<version>.tmpl and renders it with the prompt context.
func LoadSystemPromptVersion(ctx PromptContext, version string) (string, error) {
	version = strings.TrimSpace(version)
	if version == "" {
		return "", fmt.Errorf("prompt version is required")
	}

	name := "versions/" + version + ".tmpl"
	content, err := versionFS.ReadFile(name)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", name, err)
	}

	rendered, err := renderSystemTemplate(string(content), ctx)
	if err != nil {
		return "", fmt.Errorf("render %s: %w", name, err)
	}

	return rendered, nil
}
