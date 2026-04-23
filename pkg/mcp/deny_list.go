// Package mcp provides MCP integration tools.
package mcp

// DefaultDangerousTools is a hardcoded list of tool names that should
// never be allowed to execute via MCP for safety reasons.
//
// Adapted from openclaw/src/mcp/dangerous-tools.ts.
var DefaultDangerousTools = map[string]bool{
	"execute_script": true,
	"system_command": true,
	"delete_file":    true,
	"overwrite_file": true,
	"rm":             true,
	"chmod":          true,
	"chown":          true,
}

// IsDangerous returns true if the given tool name is considered hazardous.
// This operates purely on the hardcoded DefaultDangerousTools deny list.
func IsDangerous(toolName string) bool {
	return DefaultDangerousTools[toolName]
}
