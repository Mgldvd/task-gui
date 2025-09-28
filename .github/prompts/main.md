Role

You are a senior engineer and TUI designer. Refactor an existing CLI TUI into a polished, cross-platform companion for Task (go-task). Preserve the current GUI look & feel, but replace the data and execution model to be Taskfile-only.

Objective

Build a terminal UI that:

Discovers tasks exclusively from Task files (e.g., Taskfile.yml, Taskfile.yaml, Taskfile.dist.yml, and included/extended Taskfiles).

Lists all available tasks using the task name as the title.

Displays the task’s description (desc: if present).

Displays the task’s commands (cmd: entries).

Runs tasks directly from the UI by invoking the system-installed task binary.

Preserves the current GUI layout and interactions while swapping in Taskfile-based logic.

Hard Requirements (Taskfile-Only)

• The existing application code is a reference for layout and UX only—do not carry over custom command providers.
• No custom commands, no .commands, no custom.yml or other non-Taskfile sources.
• This TUI does not reimplement Task; it complements it, providing a launcher and browser for Taskfiles.
• Treat Task as a required preinstall (must be on PATH).

Official Documentation (reference only; do not scrape at runtime)

• Guide: https://taskfile.dev/docs/guide

• Getting Started: https://taskfile.dev/docs/getting-started

Inputs

• Current TUI codebase (use for visual structure, keybinds, layout).
• Taskfiles in/under the working directory (plus includes/extends).
• task binary available on PATH.

Core Requirements

Task Discovery
• Enumerate tasks from Task’s official introspection (task --list and/or JSON output if available).
• Merge includes/extends so users see the complete task set.
• Capture and display:
– name (UI title)
– desc: (if present)
– cmd: (all commands for the task)

Presentation
• Keep existing panes, colors, and keybinds.
• Search/filter by name and description.
• Sorting: name asc/desc, recently run, favorites/pinned.
• Details pane shows: name, desc, all cmd entries.

Execution
• Launch tasks via task <task-name> (plus selected vars/flags).
• Stream live output in an output pane with scrollback and copy.
• Show exit code, elapsed time; allow cancel/interrupt (SIGINT).
• Preserve current run controls, rewired to Task.

Variables / Flags
• Detect vars from Taskfiles and prompt users before execution.
• Support pass-through args (-- where applicable).
• Remember recent var values per task (session cache).

Project Discovery
• On startup, locate the nearest Taskfile from CWD upward.
• Provide a project/directory switcher.
• Manual refresh + file-watch auto-refresh to pick up Taskfile changes.

Resilience & UX
• If Task is missing, show a friendly error with install hints and keep the app stable.
• Clear empty states if no Taskfile is found.
• Graceful handling of malformed or unsupported Taskfiles—never crash the UI.

Cross-Platform
• macOS, Linux, Windows (process spawning and signal handling accounted for).
• Responsive to narrow terminals; dynamic resize support.

Performance
• Cache discovered tasks; avoid expensive reprocessing.
• Keep memory bounded; avoid unbounded log buffers.

Accessibility
• Keyboard-first operation; visible keybind hints.
• High-contrast theme option; respect terminal colors if feasible.

Testing
• Integration tests covering:
– Tasks with and without desc:
– Includes/extends merging
– No-op task execution and output capture
– Variable prompts/pass-through
• Smoke tests across supported platforms.

Acceptance Criteria

Launching the TUI in a repo with a Taskfile shows every task with name, desc, and cmds.

Running a task executes task <name> and streams live output; cancel works without crashing.

Search and sorting function correctly.

Refresh reflects Taskfile edits and includes.

If Task or Taskfile is missing, UI shows clear, actionable guidance.

Works on macOS, Linux, and Windows.

No .commands, no custom.yml, no custom sources—Taskfiles only.

Deliverables

• Fully refactored TUI sourcing only from Taskfiles (name, desc, cmds).
• Minimal usage docs describing project selection, variable prompts, and flag handling.
• Test cases and run instructions.

Final Reminder

The app is a companion to Task, not a replacement. It must display task name, desc, and cmd directly from Taskfiles, and execute via the task CLI.