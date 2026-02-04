package config

import (
	"os"
	"path/filepath"
)

type ResolvedPath struct {
	Path   string
	Scope  string
	Exists bool
}

func ResolveConfigPath() (ResolvedPath, error) {
	global := GlobalConfigPath()
	if exists(global) {
		return ResolvedPath{Path: global, Scope: ScopeGlobal, Exists: true}, nil
	}

	return ResolvedPath{Path: global, Scope: ScopeGlobal, Exists: false}, nil
}

func ResolveDbPath() (ResolvedPath, error) {
	global := GlobalDbPath()
	if exists(global) {
		return ResolvedPath{Path: global, Scope: ScopeGlobal, Exists: true}, nil
	}

	return ResolvedPath{Path: global, Scope: ScopeGlobal, Exists: false}, nil
}

func GlobalConfigPath() string {
	return filepath.Join(homeDir(), ".txp", "config.json")
}

func GlobalDbPath() string {
	return filepath.Join(homeDir(), ".txp", "txp.db")
}

func cwd() string {
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	return dir
}

func homeDir() string {
	dir, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return dir
}

func exists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}
