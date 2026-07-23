package pathnorm

import (
	"path/filepath"
	"strings"
)

// Abs normalizes a filesystem path for the current OS.
// Supports Git-Bash/Cursor style "/C:/Users/..." in addition to native absolutes.
// Relative paths are resolved from the process working directory; use AbsIn for module-relative paths.
func Abs(p string) (string, error) {
	return AbsIn("", p)
}

// AbsIn resolves p. Absolute paths (incl. /C:/...) are cleaned as-is.
// Relative paths are joined to base when base is non-empty, otherwise resolved from cwd.
func AbsIn(base, p string) (string, error) {
	p = strings.TrimSpace(p)
	p = strings.Trim(p, `"'`)
	// /C:/Users/... (MSYS/Git/Cursor paste)
	if len(p) >= 3 && p[0] == '/' && p[2] == ':' {
		p = p[1:]
	}
	p = filepath.FromSlash(p)
	p = filepath.Clean(p)
	if filepath.IsAbs(p) {
		return p, nil
	}
	if base != "" {
		return filepath.Clean(filepath.Join(base, p)), nil
	}
	return filepath.Abs(p)
}
