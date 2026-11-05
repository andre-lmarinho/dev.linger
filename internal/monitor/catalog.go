package monitor

import (
	"path/filepath"
	"strings"
)

type signature struct {
	entrypoint     string
	name           string
	executables    []string
	arguments      []string
	commands       []string
	defaultCommand bool
}

var developmentEntrypoints = []signature{
	{entrypoint: "/node_modules/next/dist/bin/next", name: "Next.js", executables: []string{"node", "nodejs"}, commands: []string{"dev"}},
	{entrypoint: "/node_modules/astro/astro.js", name: "Astro", executables: []string{"node", "nodejs"}, commands: []string{"dev"}},
	{entrypoint: "/node_modules/astro/bin/astro.mjs", name: "Astro", executables: []string{"node", "nodejs"}, commands: []string{"dev"}},
	{entrypoint: "/node_modules/vite/bin/vite.js", name: "Vite", executables: []string{"node", "nodejs"}, commands: []string{"dev", "serve"}, defaultCommand: true},
}

func catalog(args []string, cwd string) string {
	return matchCatalog(args, cwd, developmentEntrypoints)
}

func matchCatalog(args []string, cwd string, entries []signature) string {
	if len(args) == 0 || cwd == "" {
		return ""
	}
	for _, entry := range entries {
		i := 0
		for _, executable := range entry.executables {
			if filepath.Base(args[0]) == executable {
				i = 1
				break
			}
		}
		if entry.entrypoint == "" && i == 0 {
			continue
		}
		if entry.entrypoint != "" {
			if i >= len(args) {
				continue
			}
			path := args[i]
			if !filepath.IsAbs(path) {
				path = filepath.Join(cwd, path)
			}
			if resolved, err := filepath.EvalSymlinks(path); err == nil {
				path = resolved
			}
			path = filepath.ToSlash(filepath.Clean(path))
			if !strings.HasSuffix(path, entry.entrypoint) {
				continue
			}
			i++
		}
		matched := true
		for _, arg := range entry.arguments {
			if i >= len(args) || args[i] != arg {
				matched = false
				break
			}
			i++
		}
		if !matched {
			continue
		}
		if len(entry.commands) == 0 && !entry.defaultCommand {
			return entry.name
		}
		if i == len(args) || strings.HasPrefix(args[i], "-") {
			if entry.defaultCommand {
				return entry.name
			}
			continue
		}
		for _, command := range entry.commands {
			if args[i] == command {
				return entry.name
			}
		}
	}
	return ""
}
