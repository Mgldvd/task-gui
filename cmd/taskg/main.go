package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"taskg/internal/app"
	"taskg/internal/taskmeta"
	"taskg/internal/version"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var (
	theme      string
	noMouse    bool
	projectDir string
)

var rootCmd = &cobra.Command{
	Use:   "taskg",
	Short: "Task Runner GUI: browse and run Taskfile tasks (companion UI for go-task)",
	Long: `Task Runner GUI is a terminal user interface that discovers tasks from Taskfiles (including includes/extends)
and lets you search, inspect, and run them. It requires the 'task' binary to be installed and on PATH.`,
	Version: version.Version,
	Run: func(cmd *cobra.Command, args []string) {
		// Determine working directory / project root
		startDir := projectDir
		if startDir == "" {

			cwd, _ := os.Getwd()
			startDir = cwd
		}
		root, err := taskmeta.FindNearestTaskfileRoot(startDir)
		var tasks []taskmeta.Task
		var model *app.TaskModel
		if err != nil {
			model = app.NewTaskModel(nil, theme, !noMouse, filepath.Base(startDir))
			model.Error("No Taskfile found in this or parent directories. Use --project to point elsewhere or create a Taskfile.yml.")
		} else {
			tasks, err = taskmeta.DiscoverTasks(root)
			if err != nil {
				model = app.NewTaskModel(nil, theme, !noMouse, filepath.Base(root))
				model.SetProjectRoot(root)
				model.Error(fmt.Sprintf("Failed to enumerate tasks: %v", err))
			} else if len(tasks) == 0 {
				model = app.NewTaskModel(nil, theme, !noMouse, filepath.Base(root))
				model.SetProjectRoot(root)
				model.Error("No tasks discovered in Taskfile.")
			} else {
				model = app.NewTaskModel(tasks, theme, !noMouse, filepath.Base(root))
				model.SetProjectRoot(root)
			}
		}
		var options []tea.ProgramOption
		options = append(options, tea.WithAltScreen())
		if !noMouse {
			options = append(options, tea.WithMouseCellMotion())
		}
		p := tea.NewProgram(model, options...)
		finalModel, errRun := p.Run()
		if errRun != nil {
			log.Fatalf("Failed to run app: %v", errRun)
		}
		// After TUI exits, check if a task should be run
		if m, ok := finalModel.(*app.TaskModel); ok {
			if m.ShouldRun() {
				taskCmd := m.TaskToRun()
				// Clear the screen for better visibility
				fmt.Print("\033[H\033[2J")
				fmt.Println()

				if len(taskCmd) == 0 {
					fmt.Fprintln(os.Stderr, "No task selected. Please select a valid task.")
					return
				}

				taskName := taskCmd[0]
				taskArgs := taskCmd[1:]

				argsForExec := []string{taskName}
				if len(taskArgs) > 0 {
					argsForExec = append(argsForExec, taskArgs...)
				}

				c := exec.Command("task", argsForExec...)
				if root != "" {
					c.Dir = root
				}
				c.Stdout = os.Stdout
				c.Stderr = os.Stderr
				c.Stdin = os.Stdin
				if err := c.Run(); err != nil {
					// The task exiting with a non-zero status is not necessarily an
					// error in the GUI runner, so just log it.
					fmt.Fprintf(os.Stderr, "Task exited: %v\n", err)
				}
			}
		}
	},
}

func init() {
	rootCmd.Flags().StringVar(&theme, "theme", "dark", "Theme: dark or light")
	rootCmd.Flags().BoolVar(&noMouse, "no-mouse", false, "Disable mouse support")
	rootCmd.Flags().StringVar(&projectDir, "project", "", "Start directory for locating nearest Taskfile (defaults to CWD)")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
