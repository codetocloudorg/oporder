#!/usr/bin/env node
// The `oporder` command this package registers — honestly a placeholder,
// not the real CLI. Exits non-zero so scripts calling it don't mistake
// this for success, same discipline as the Homebrew tap's placeholder
// formula calling odie() instead of silently no-opping.
require("../lib/message")();
process.exitCode = 1;
