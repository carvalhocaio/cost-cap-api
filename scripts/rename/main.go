// Command rename renames the Go module and CLI binary of this template.
//
// Usage:
//
//	go run ./scripts/rename <new/module/path>
package main

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	modulePathRe = regexp.MustCompile(`^[a-zA-Z0-9._~-]+(/[a-zA-Z0-9._~-]+)*$`)
	binNameRe    = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
)

// textFiles lists non-Go files (relative to the root) that reference the names.
var textFiles = []string{"Makefile", "README.md", ".golangci.yml"}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: go run ./scripts/rename <new/module/path>")
		os.Exit(1)
	}
	if err := rename(".", os.Args[1]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func rename(rootDir, rawModule string) error {
	newModule := strings.TrimSpace(rawModule)
	if newModule == "" {
		return errors.New("module path cannot be empty")
	}
	if !modulePathRe.MatchString(newModule) {
		return fmt.Errorf("invalid module path %q", newModule)
	}
	newName := path.Base(newModule)
	if !binNameRe.MatchString(newName) {
		return fmt.Errorf("binary name %q (last path segment) can only contain "+
			"letters, numbers, hyphens, and underscores", newName)
	}

	oldModule, err := readModulePath(filepath.Join(rootDir, "go.mod"))
	if err != nil {
		return err
	}
	oldName, err := currentBinName(filepath.Join(rootDir, "cmd"))
	if err != nil {
		return err
	}

	fmt.Printf("Renaming module '%s' -> '%s'...\n", oldModule, newModule)
	fmt.Printf("Renaming binary '%s' -> '%s'...\n", oldName, newName)

	if oldName != newName {
		oldDir := filepath.Join(rootDir, "cmd", oldName)
		newDir := filepath.Join(rootDir, "cmd", newName)
		if _, err := os.Stat(newDir); err == nil {
			return fmt.Errorf("target directory %q already exists", newDir)
		}
		if err := os.Rename(oldDir, newDir); err != nil {
			return err
		}
	}

	nameRe := regexp.MustCompile(`\b` + regexp.QuoteMeta(oldName) + `\b`)
	replace := func(content string) string {
		content = strings.ReplaceAll(content, oldModule, newModule)
		return nameRe.ReplaceAllString(content, newName)
	}

	files, err := goFiles(rootDir)
	if err != nil {
		return err
	}
	files = append(files, filepath.Join(rootDir, "go.mod"))
	for _, f := range textFiles {
		files = append(files, filepath.Join(rootDir, f))
	}
	for _, f := range files {
		if err := rewriteFile(f, replace); err != nil {
			return err
		}
	}

	fmt.Println("Running go mod tidy...")
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = rootDir
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go mod tidy: %w", err)
	}

	fmt.Printf("\nProject successfully renamed to '%s' (binary: '%s')!\n", newModule, newName)
	return nil
}

func readModulePath(goMod string) (string, error) {
	f, err := os.Open(goMod) //nolint:gosec // path is built from the repo root
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if mod, ok := strings.CutPrefix(strings.TrimSpace(scanner.Text()), "module "); ok {
			return strings.Trim(strings.TrimSpace(mod), `"`), nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", errors.New("could not find module directive in go.mod")
}

func currentBinName(cmdDir string) (string, error) {
	entries, err := os.ReadDir(cmdDir)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if e.IsDir() {
			return e.Name(), nil
		}
	}
	return "", errors.New("could not find current binary in cmd/")
}

func goFiles(rootDir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(rootDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && (d.Name() == ".git" || d.Name() == "bin") {
			return filepath.SkipDir
		}
		if !d.IsDir() && strings.HasSuffix(p, ".go") {
			files = append(files, p)
		}
		return nil
	})
	return files, err
}

func rewriteFile(file string, replace func(string) string) error {
	info, err := os.Stat(file)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	data, err := os.ReadFile(file) //nolint:gosec // path is built from the repo root
	if err != nil {
		return err
	}
	updated := replace(string(data))
	if updated == string(data) {
		return nil
	}
	return os.WriteFile(file, []byte(updated), info.Mode().Perm())
}
