# Design: @file Support for opencode_skill

## Problem Statement

Current `opencode_skill` only accepts messages as command-line arguments:
```bash
opencode_skill --sync proj sess "Short message here"
```

This fails for multi-line, complex prompts passed from AI agents. The original approach using heredoc with `/dev/stdin` failed because the skill treated `/dev/stdin` as a literal string.

## Solution: @file Syntax

Support reading message content from a file when message starts with `@`:

```bash
opencode_skill --sync proj sess @/path/to/task.txt
```

## Implementation

### 1. Add helper function in main.go

```go
// resolveMessage handles @file or normal message args
func resolveMessage(messageParts []string) (string, error) {
    if len(messageParts) == 0 {
        return "", errors.New("no message provided")
    }
    
    first := messageParts[0]
    
    // @file syntax - read content from file
    if strings.HasPrefix(first, "@") {
        filename := first[1:] // Remove @
        content, err := os.ReadFile(filename)
        if err != nil {
            return "", fmt.Errorf("failed to read file %s: %w", filename, err)
        }
        return strings.TrimSpace(string(content)), nil
    }
    
    // Default: join args with space (backward compatible)
    return strings.Join(messageParts, " "), nil
}
```

### 2. Modify prompt handling in main()

Around line 246-249, replace:
```go
if len(messageParts) == 0 {
    fmt.Println("No message provided.")
    return
}

fullMessage := strings.Join(messageParts, " ")
```

With:
```go
fullMessage, err := resolveMessage(messageParts)
if err != nil {
    fmt.Printf("Error: %v\n", err)
    return
}
```

### 3. Update usage documentation

In `printUsage()`, update the example:
```
  opencode_skill [flags] <PROJECT> <SESSION_NAME> <MESSAGE>
  opencode_skill [flags] <PROJECT> <SESSION_NAME> @file.txt  # Read message from file
```

## Usage Examples

```bash
# Write task to file first
cat > /tmp/task.txt << 'EOF'
You are implementing Tasks 6-9 of a peak-hour auto-switch feature for an LLM proxy.
[full task description]
EOF

# Invoke with @file
opencode_skill --sync llmproxy db-foundation @/tmp/task.txt
```

## Files to Modify

| File | Change |
|------|--------|
| `opencode-web-skill/opencode_skill_src/main.go` | Add `resolveMessage()` function; update prompt handling (lines 246-249); update usage text |
| `opencode-web-skill/SKILL.md` | Add documentation for @file syntax |

## SKILL.md Update

Add new section after "Send Commands" to document @file syntax:

```markdown
### Long Message Input via @file

For long or complex prompts, use `@file` syntax to read from a file:

**Path format:** `/tmp/opencode_skill/input_files/{project_name}/{session_name}_input.txt`

**Example:**
```bash
# Create input file with the task
cat > /tmp/opencode_skill/input_files/myapp/feature-login_input.txt << 'EOF'
Implement user login with email and password.
Include session management and logout functionality.
EOF

# Send using @file syntax
opencode_skill --sync myapp feature-login @/tmp/opencode_skill/input_files/myapp/feature-login_input.txt
```

## Dependencies

None - uses only standard library:
- `os.ReadFile` for file reading
- `strings.TrimSpace` for trimming
- `fmt.Errorf` for error formatting

## Backward Compatibility

- Existing usage `opencode_skill proj sess "message"` continues to work unchanged
- The `@` prefix is unlikely to be used as an actual message start
- No changes to daemon protocol or API