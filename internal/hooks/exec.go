package hooks

import "os/exec"

// Indirected so tests can swap it. Kept minimal since hooks.Run does the
// streaming/env wiring itself.
var execCommand = exec.Command
