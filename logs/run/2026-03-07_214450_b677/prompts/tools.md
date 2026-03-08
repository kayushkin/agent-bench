# Tool Definitions

11 tools registered

## 1. shell

Execute a shell command via bash -c. Returns stdout+stderr combined. Use for running programs, git, builds, etc.

**Schema:**

```json
{
  "properties": {
    "command": {
      "description": "Shell command to execute",
      "type": "string"
    },
    "workdir": {
      "description": "Working directory (optional, defaults to cwd)",
      "type": "string"
    }
  },
  "required": [
    "command"
  ],
  "type": "object"
}
```

## 2. read_file

Read the contents of a file. For large files, use offset (1-indexed line number) and limit (max lines) to read a portion.

**Schema:**

```json
{
  "properties": {
    "limit": {
      "description": "Maximum number of lines to return (optional)",
      "type": "integer"
    },
    "offset": {
      "description": "Line number to start from (1-indexed, optional)",
      "type": "integer"
    },
    "path": {
      "description": "Path to the file to read",
      "type": "string"
    }
  },
  "required": [
    "path"
  ],
  "type": "object"
}
```

## 3. write_file

Create or overwrite a file with the given content. Creates parent directories automatically.

**Schema:**

```json
{
  "properties": {
    "content": {
      "description": "Content to write to the file",
      "type": "string"
    },
    "path": {
      "description": "Path to the file to write",
      "type": "string"
    }
  },
  "required": [
    "path",
    "content"
  ],
  "type": "object"
}
```

## 4. edit_file

Edit a file by replacing an exact text match with new text. The old_text must match exactly (including whitespace). Use for precise, surgical edits.

**Schema:**

```json
{
  "properties": {
    "new_text": {
      "description": "New text to replace the old text with",
      "type": "string"
    },
    "old_text": {
      "description": "Exact text to find and replace",
      "type": "string"
    },
    "path": {
      "description": "Path to the file to edit",
      "type": "string"
    }
  },
  "required": [
    "path",
    "old_text",
    "new_text"
  ],
  "type": "object"
}
```

## 5. list_files

List files and directories at a path. Use recursive=true for a tree listing (respects .gitignore patterns).

**Schema:**

```json
{
  "properties": {
    "path": {
      "description": "Directory path to list",
      "type": "string"
    },
    "recursive": {
      "description": "List recursively (default: false)",
      "type": "boolean"
    }
  },
  "required": [
    "path"
  ],
  "type": "object"
}
```

## 6. memory_search

Search persistent memories by semantic similarity to a query. Returns relevant memories ranked by similarity, importance, and recency.

**Schema:**

```json
{
  "properties": {
    "limit": {
      "description": "Maximum number of results to return (default: 10)",
      "type": "integer"
    },
    "query": {
      "description": "Search query text",
      "type": "string"
    }
  },
  "required": [
    "query"
  ],
  "type": "object"
}
```

## 7. memory_save

Store a new memory for persistent recall across sessions. Memories are automatically embedded for semantic search.

**Schema:**

```json
{
  "properties": {
    "content": {
      "description": "The memory content to store",
      "type": "string"
    },
    "importance": {
      "description": "Importance score 0-1 (default: 0.5). Higher scores = higher priority in search.",
      "type": "number"
    },
    "source": {
      "description": "Source of the memory: 'user', 'agent', 'system' (default: 'agent')",
      "type": "string"
    },
    "tags": {
      "description": "Tags for categorization (e.g., 'code', 'preference', 'fact')",
      "items": {
        "type": "string"
      },
      "type": "array"
    }
  },
  "required": [
    "content"
  ],
  "type": "object"
}
```

## 8. memory_expand

Retrieve the full content of a memory by ID. Useful for expanding compacted summaries or revisiting specific memories.

**Schema:**

```json
{
  "properties": {
    "id": {
      "description": "Memory ID to retrieve",
      "type": "string"
    }
  },
  "required": [
    "id"
  ],
  "type": "object"
}
```

## 9. memory_forget

Mark a memory as forgotten/irrelevant. This is a soft delete — the memory remains in storage but won't appear in search results.

**Schema:**

```json
{
  "properties": {
    "id": {
      "description": "Memory ID to forget",
      "type": "string"
    }
  },
  "required": [
    "id"
  ],
  "type": "object"
}
```

## 10. repo_map

Generate a structural map of the codebase showing packages, functions, types, and file organization. Use this to understand the project structure without reading full files.

**Schema:**

```json
{
  "properties": {
    "format": {
      "description": "Output format: 'compact' (default, abbreviated) or 'full' (complete signatures).",
      "type": "string"
    },
    "path": {
      "description": "Subdirectory to map (relative to repo root). Leave empty for entire repo.",
      "type": "string"
    }
  },
  "type": "object"
}
```

## 11. recent_files

List files that were recently modified, with metadata like line count, modification time, and importance score. Use this to see what's been actively worked on.

**Schema:**

```json
{
  "properties": {
    "include_content": {
      "description": "If true, include file contents in addition to metadata. Default: false (metadata only)",
      "type": "boolean"
    },
    "since": {
      "description": "Time window to search (e.g., '2h', '1d', '7d'). Default: '24h'",
      "type": "string"
    }
  },
  "type": "object"
}
```

---
**Total:** 11 tools, ~1921 tokens
