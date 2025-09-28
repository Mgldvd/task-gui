package taskmeta

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Task represents a discovered task from Taskfile
type Task struct {
	Name string
	Desc string
	Cmds []string // flattened list of command lines extracted from task definition
	Line int      // line number in the taskfile for preserving file order
	// Future: Vars []string, Sources []string, etc.
}

// listJSON models a subset of `task --list --json` output. We only capture what we need.
// The task CLI (as of Task v3) returns something akin to:
// {"tasks":[{"name":"build","desc":"Build the project"}, ...]}
type listJSON struct {
	Tasks []struct {
		Name     string `json:"name"`
		Desc     string `json:"desc"`
		Location struct {
			Line int `json:"line"`
		} `json:"location"`
	} `json:"tasks"`
}

// rawYAMLTask minimal structure for parsing Taskfile directly when JSON list unavailable.
type rawYAMLTask struct {
	Desc string `yaml:"desc"`
	Cmds any    `yaml:"cmds"` // can be string or []string or []interface{}
	Cmd  any    `yaml:"cmd"`  // alias
	// We intentionally ignore other Taskfile keys for now.
}

// taskfileRootCandidates names we consider as Taskfile roots.
var taskfileRootCandidates = []string{
	"Taskfile.yml", "Taskfile.yaml", "Taskfile.dist.yml", "Taskfile.dist.yaml",
	"taskfile.yml", "taskfile.yaml", "taskfile.dist.yml", "taskfile.dist.yaml",
}

// FindNearestTaskfileRoot walks upward from start until it finds a Taskfile.* returning that directory.
func FindNearestTaskfileRoot(start string) (string, error) {
	dir := start
	for {
		for _, name := range taskfileRootCandidates {
			if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
				return dir, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", errors.New("no Taskfile found in parent directories")
}

// DiscoverTasks returns all tasks available (merged includes handled by task CLI itself).
// Strategy:
// 1. Run `task --list --json` in root (preferred)
// 2. If that fails (older task?), run `task --list` and parse lines `* name: desc`
// 3. As a final fallback, parse the Taskfile YAML minimally for top-level tasks map.
func DiscoverTasks(root string) ([]Task, error) {
	if root == "" {
		cwd, _ := os.Getwd()
		root = cwd
	}

	// Ensure task binary exists early
	if _, err := exec.LookPath("task"); err != nil {
		return nil, fmt.Errorf("task binary not found in PATH: %w", err)
	}

	// Preferred: JSON list (gives names & desc only)
	tasks, err := listViaJSON(root)
	if err == nil && len(tasks) > 0 {
		// Enrich with command lines by parsing Taskfile YAML (optional best effort)
		enrichTaskCmds(root, tasks)
		return tasks, nil
	}

	// Fallback: parse `task --list` plain text
	tasks, errPlain := listViaPlain(root)
	if errPlain == nil && len(tasks) > 0 {
		enrichTaskCmds(root, tasks)
		return tasks, nil
	}

	// Last resort: parse YAML directly (top-level tasks only)
	tasks, errY := parseTaskfileYAML(root)
	if errY == nil && len(tasks) > 0 {
		return tasks, nil
	}

	// Compose meaningful error chain
	return nil, fmt.Errorf("failed to discover tasks (json:%v plain:%v yaml:%v)", err, errPlain, errY)
}

func listViaJSON(root string) ([]Task, error) {
	cmd := exec.Command("task", "--list", "--json")
	cmd.Dir = root
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	var lj listJSON
	if err := json.Unmarshal(out.Bytes(), &lj); err != nil {
		return nil, err
	}
	var tasks []Task
	for _, t := range lj.Tasks {
		tasks = append(tasks, Task{Name: t.Name, Desc: t.Desc, Line: t.Location.Line})
	}
	return tasks, nil
}

func listViaPlain(root string) ([]Task, error) {
	cmd := exec.Command("task", "--list")
	cmd.Dir = root
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	lines := strings.Split(out.String(), "\n")
	var tasks []Task
	for _, l := range lines {
		l = strings.TrimSpace(l)
		// Typical line format: * build: Build the project (desc optional)
		if strings.HasPrefix(l, "*") {
			l = strings.TrimPrefix(l, "*")
			l = strings.TrimSpace(l)
			// split at first ':'
			name := l
			desc := ""
			if idx := strings.Index(l, ":"); idx != -1 {
				name = strings.TrimSpace(l[:idx])
				desc = strings.TrimSpace(l[idx+1:])
			}
			if name != "" {
				tasks = append(tasks, Task{Name: name, Desc: desc})
			}
		}
	}
	return tasks, nil
}

// parseTaskfileYAML best-effort parse top-level tasks to capture desc & cmds for fallback.
func parseTaskfileYAML(root string) ([]Task, error) {
	// choose first existing candidate
	var path string
	for _, c := range taskfileRootCandidates {
		if _, err := os.Stat(filepath.Join(root, c)); err == nil {
			path = filepath.Join(root, c)
			break
		}
	}
	if path == "" {
		return nil, errors.New("no Taskfile found")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var node map[string]any
	if err := yaml.Unmarshal(data, &node); err != nil {
		return nil, err
	}
	// tasks section may be map[string]any
	section, ok := node["tasks"].(map[string]any)
	if !ok {
		return nil, errors.New("no tasks map in Taskfile")
	}
	var tasks []Task
	for name, raw := range section {
		rm, _ := raw.(map[string]any)
		if rm == nil {
			continue
		}
		var tsk Task
		tsk.Name = name
		if d, ok := rm["desc"].(string); ok {
			tsk.Desc = d
		}
		// commands may be in cmds or cmd
		if v, ok := rm["cmds"]; ok {
			tsk.Cmds = extractCmds(v)
		}
		if len(tsk.Cmds) == 0 {
			if v, ok := rm["cmd"]; ok {
				tsk.Cmds = extractCmds(v)
			}
		}
		tasks = append(tasks, tsk)
	}
	return tasks, nil
}

func extractCmds(v any) []string {
	var out []string
	switch vv := v.(type) {
	case string:
		if strings.TrimSpace(vv) != "" {
			out = append(out, vv)
		}
	case []any:
		for _, it := range vv {
			out = append(out, extractCmds(it)...)
		}
	case []string:
		for _, s := range vv {
			out = append(out, s)
		}
	}
	return out
}

// enrichTaskCmds attempts to parse Taskfile YAML to attach command lines for detail view.
func enrichTaskCmds(root string, tasks []Task) {
	// Build index for quick update
	idx := make(map[string]*Task, len(tasks))
	for i := range tasks {
		idx[tasks[i].Name] = &tasks[i]
	}
	parsed, err := parseTaskfileYAML(root)
	if err != nil {
		return
	}
	for _, p := range parsed {
		if t, ok := idx[p.Name]; ok {
			if len(t.Cmds) == 0 && len(p.Cmds) > 0 {
				t.Cmds = p.Cmds
			}
			if t.Desc == "" && p.Desc != "" {
				t.Desc = p.Desc
			}
		}
	}
}
