# New System Prompt

Prompt revisions are plain files in `prompts/versions/`.  
Selection is a code constant (`systemPromptVersion` in `main.go`).

## Code Anchors

- Version selector: `main.go`
- Loader + template rendering: `prompts/prompts.go`
- Revisions: `prompts/versions/*.tmpl`

## Expected Workflow

- Add a new versioned `.tmpl` file.
- Point `systemPromptVersion` at it.
- Run and validate startup/runtime behavior.

## Template Contract

Templates are rendered from `HostContext` with strict key checking.

Current placeholders:
- `{{.CWD}}`
- `{{.OS}}`
- `{{.Arch}}`
- `{{.Username}}`

## Extending Placeholders

If a template needs a new key, evolve the contract in one pass:
- Add the field to `HostContext`.
- Populate it where `hostCtx` is built in `main.go`.
- Use it in the `.tmpl`.

Keep additions explicit and small. Treat template keys as a stable interface.

## Failure Signals

- Missing version file: loader read error.
- Unknown template key: render error (`missingkey=error`).
