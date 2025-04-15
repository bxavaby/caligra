// BYZRA ⸻ internal/config/profile.go
// profile handling for metadata injection

package config

import (
	"fmt"
	"os"
	"path/filepath"

	lua "github.com/yuin/gopher-lua"
)

// metadata profile to inject
type Profile struct {
	Author       string
	Software     string
	Created      string
	Organization string
	Location     string
	Comment      string
}

// loads profile
func LoadProfile() (map[string]string, error) {
	// search common locations
	paths := []string{
		"config/profile.lua",
		"./profile.lua",
		filepath.Join(os.Getenv("HOME"), ".caligra/config/profile.lua"),
	}

	var profilePath string
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			profilePath = path
			break
		}
	}

	if profilePath == "" {
		return nil, fmt.Errorf("profile.lua not found in search paths")
	}

	data, err := os.ReadFile(profilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read profile: %w", err)
	}

	L := lua.NewState()
	defer L.Close()

	if err := L.DoString(string(data)); err != nil {
		return nil, fmt.Errorf("failed to execute profile Lua: %w", err)
	}

	result := L.Get(-1)
	if result.Type() != lua.LTTable {
		return nil, fmt.Errorf("profile Lua must return a table")
	}

	// convert Lua table 2 Go map
	profile := make(map[string]string)
	lTable := result.(*lua.LTable)
	lTable.ForEach(func(k, v lua.LValue) {
		if k.Type() == lua.LTString && v.Type() == lua.LTString {
			profile[k.String()] = v.String()
		}
	})

	// validate required fields
	requiredFields := []string{"author", "software", "created"}
	for _, field := range requiredFields {
		if _, ok := profile[field]; !ok {
			return nil, fmt.Errorf("profile is missing required field: %s", field)
		}
	}

	return profile, nil
}

// fallback values if no profile is found
func GetDefaultProfile() map[string]string {
	return map[string]string{
		"author":       "nynynn",
		"software":     "liberty/1.0",
		"created":      "2000-01-01",
		"organization": "none",
		"location":     "unknown",
		"comment":      "sanitized",
	}
}
