package monitor

import (
	"path/filepath"
	"slices"
	"strings"
)

type signature struct {
	entrypoint     string
	name           string
	executables    []string
	arguments      []string
	commands       []string
	defaultCommand bool
	requireValue   bool
}

var developmentEntrypoints = []signature{
	{entrypoint: "/node_modules/next/dist/bin/next", name: "Next.js", executables: []string{"node", "nodejs"}, commands: []string{"dev"}},
	{entrypoint: "/node_modules/astro/astro.js", name: "Astro", executables: []string{"node", "nodejs"}, commands: []string{"dev"}},
	{entrypoint: "/node_modules/astro/bin/astro.mjs", name: "Astro", executables: []string{"node", "nodejs"}, commands: []string{"dev"}},
	{entrypoint: "/node_modules/vite/bin/vite.js", name: "Vite", executables: []string{"node", "nodejs"}, commands: []string{"dev", "serve"}, defaultCommand: true},
	{entrypoint: "/node_modules/nuxt/bin/nuxt.mjs", name: "Nuxt", executables: []string{"node", "nodejs"}, commands: []string{"dev"}},
	{entrypoint: "/node_modules/nuxi/bin/nuxi.mjs", name: "Nuxt", executables: []string{"node", "nodejs"}, commands: []string{"dev"}},
	{entrypoint: "/node_modules/@nuxt/cli/bin/nuxi.mjs", name: "Nuxt", executables: []string{"node", "nodejs"}, commands: []string{"dev"}},
	{entrypoint: "/node_modules/@angular/cli/bin/ng.js", name: "Angular", executables: []string{"node", "nodejs"}, commands: []string{"serve"}},
	{entrypoint: "/node_modules/@nestjs/cli/bin/nest.js", name: "NestJS", executables: []string{"node", "nodejs"}, commands: []string{"start"}},
	{entrypoint: "/node_modules/@remix-run/dev/dist/cli.js", name: "Remix", executables: []string{"node", "nodejs"}, commands: []string{"dev"}},
	{entrypoint: "/node_modules/storybook/dist/bin/dispatcher.js", name: "Storybook", executables: []string{"node", "nodejs"}, commands: []string{"dev"}},
	{entrypoint: "/node_modules/@storybook/cli/dist/bin/index.js", name: "Storybook", executables: []string{"node", "nodejs"}, commands: []string{"dev"}},
	{entrypoint: "/node_modules/expo/bin/cli", name: "Expo", executables: []string{"node", "nodejs"}, commands: []string{"start"}},
	{name: "Python HTTP server", executables: []string{"python", "python3", "python3.*"}, arguments: []string{"-m", "http.server"}},
	{name: "Django", executables: []string{"python", "python3", "python3.*"}, arguments: []string{"-m", "django", "runserver"}},
	{entrypoint: "/manage.py", name: "Django", executables: []string{"python", "python3", "python3.*"}, commands: []string{"runserver"}},
	{name: "Flask", executables: []string{"python", "python3", "python3.*"}, arguments: []string{"-m", "flask"}, commands: []string{"run"}},
	{entrypoint: "/bin/flask", name: "Flask", executables: []string{"python", "python3", "python3.*"}, commands: []string{"run"}},
	{name: "Uvicorn", executables: []string{"python", "python3", "python3.*"}, arguments: []string{"-m", "uvicorn"}, requireValue: true},
	{entrypoint: "/bin/uvicorn", name: "Uvicorn", executables: []string{"python", "python3", "python3.*"}, requireValue: true},
	{name: "Gunicorn", executables: []string{"python", "python3", "python3.*"}, arguments: []string{"-m", "gunicorn"}, requireValue: true},
	{entrypoint: "/bin/gunicorn", name: "Gunicorn", executables: []string{"python", "python3", "python3.*"}, requireValue: true},
	{name: "PHP server", executables: []string{"php"}, arguments: []string{"-S"}, requireValue: true},
}

func catalog(args []string, cwd string) string {
	return matchCatalog(args, cwd, developmentEntrypoints)
}

func matchCatalog(args []string, cwd string, entries []signature) string {
	if len(args) == 0 || cwd == "" {
		return ""
	}
	// Each signature considers either the executable or its first argument.
	var paths [2]string
	var resolvedPaths [2]bool
	for _, entry := range entries {
		i := 0
		for _, executable := range entry.executables {
			if matched, _ := filepath.Match(executable, filepath.Base(args[0])); matched {
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
			if !resolvedPaths[i] {
				path := args[i]
				if !filepath.IsAbs(path) {
					path = filepath.Join(cwd, path)
				}
				if resolved, err := filepath.EvalSymlinks(path); err == nil {
					path = resolved
				}
				paths[i] = filepath.ToSlash(filepath.Clean(path))
				resolvedPaths[i] = true
			}
			if !strings.HasSuffix(paths[i], entry.entrypoint) {
				continue
			}
			i++
		}
		if len(args)-i < len(entry.arguments) || !slices.Equal(args[i:i+len(entry.arguments)], entry.arguments) {
			continue
		}
		i += len(entry.arguments)
		beforeSeparator := args[i:]
		if end := slices.Index(beforeSeparator, "--"); end >= 0 {
			beforeSeparator = beforeSeparator[:end]
		}
		if slices.Contains(beforeSeparator, "--help") || slices.Contains(beforeSeparator, "--version") {
			continue
		}
		if entry.requireValue && (i >= len(args) || args[i] == "" || strings.HasPrefix(args[i], "-")) {
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
		if slices.Contains(entry.commands, args[i]) {
			return entry.name
		}
	}
	return ""
}
