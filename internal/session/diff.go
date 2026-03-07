package session

import (
	"fmt"
	"strings"
)

func RenderDiffPreview(preview DiffPreviewEvent) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📄 %s (before) → %s (after)\n", preview.RelPath, preview.RelPath))

	oldLines := splitLines(preview.OldContent)
	newLines := splitLines(preview.NewContent)
	maxLines := len(oldLines)
	if len(newLines) > maxLines {
		maxLines = len(newLines)
	}

	for i := 0; i < maxLines; i++ {
		var oldLine, newLine string
		if i < len(oldLines) {
			oldLine = oldLines[i]
		}
		if i < len(newLines) {
			newLine = newLines[i]
		}

		switch {
		case oldLine == newLine:
			if oldLine != "" {
				sb.WriteString("  " + oldLine + "\n")
			}
		case oldLine == "":
			sb.WriteString("+ " + newLine + "\n")
		case newLine == "":
			sb.WriteString("- " + oldLine + "\n")
		default:
			sb.WriteString("- " + oldLine + "\n")
			sb.WriteString("+ " + newLine + "\n")
		}
	}

	return strings.TrimRight(sb.String(), "\n")
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	s = strings.ReplaceAll(s, "\r\n", "\n")
	lines := strings.Split(s, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}
