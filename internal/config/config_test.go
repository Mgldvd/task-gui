//go:build ignore

// Archived legacy config tests placeholder (ignored via build tag).
// Historical content moved to _archive/config/.
// Retained only because deletion encountered tooling limitation.
package ignore

// (No test code; intentionally blank.)

	tests := []struct {
		query    string
		expected int
	}{
		{"", 3},            // No filter, all commands
		{"test", 2},        // Should match first two
		{"password", 1},    // Should match third
		{"pwgen", 1},       // Should match command
		{"nonexistent", 0}, // Should match nothing
	}

	for _, test := range tests {
		filtered := FilterCommands(commands, test.query)
		if len(filtered) != test.expected {
			t.Errorf("Query '%s': expected %d results, got %d", test.query, test.expected, len(filtered))
		}
	}
}

func TestFilterCommandsCaseInsensitive(t *testing.T) {
	commands := []Command{
		{Title: "Test Command", Cmd: "echo HELLO"},
	}

	// Test case insensitive matching
	filtered := FilterCommands(commands, "TEST")
	if len(filtered) != 1 {
		t.Error("Expected case-insensitive matching to work")
	}

	filtered = FilterCommands(commands, "hello")
	if len(filtered) != 1 {
		t.Error("Expected case-insensitive matching on command to work")
	}
}
