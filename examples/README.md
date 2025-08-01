# Claude Code SDK Go Examples

This directory contains examples demonstrating how to use the Claude Code SDK for Go.

## Examples

### 1. Simple Usage (`simple/`)
Basic examples showing:
- One-shot queries
- Streaming responses
- Interactive sessions

```bash
cd simple
go run main.go
```

### 2. SDK Self-Analysis (`analyze-sdk/`)
An example that uses Claude to analyze the SDK's own code:
- Reviews the architecture
- Evaluates code quality
- Suggests improvements

```bash
cd analyze-sdk
go run main.go
```

### 3. Comment Improvement (`improve-comment/`)
Demonstrates Claude Code's ability to modify code:
- Searches for a comment that could be improved
- Explains why it needs improvement  
- Uses the Edit tool to improve the comment
- Shows system messages and tool usage

```bash
cd improve-comment
go run main.go
```

This example uses `PermissionModeAcceptEdits` to automatically accept file edits.

### 4. README Review (`review-readmes/`)
Reviews and updates all README files in the project:
- Checks code examples for accuracy
- Verifies package names and imports
- Fixes outdated information
- Improves clarity and completeness
- Shows real-time progress with streaming

```bash
cd review-readmes
go run main.go
```

This example demonstrates comprehensive documentation review with multiple file edits.

## Prerequisites

Before running any example:

1. Install Node.js
2. Install Claude Code CLI:
   ```bash
   npm install -g @anthropic-ai/claude-code
   ```
3. Ensure you have valid Claude API credentials configured

## Running Examples

From the example directory:
```bash
go run main.go
```

Or from the project root:
```bash
go run examples/simple/main.go
go run examples/analyze-sdk/main.go
```