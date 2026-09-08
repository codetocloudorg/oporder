#!/usr/bin/env node
// The `oporder` command this package registers. Execs the real platform
// binary postinstall.js downloaded next to this file, forwarding args,
// stdio, and exit code — this file itself is never the implementation,
// just a launcher, so `npm install -g oporder` and downloading the
// GitHub Release binary directly stay the same underlying program.
const fs = require("fs");
const path = require("path");
const { spawnSync } = require("child_process");

const binName = process.platform === "win32" ? "oporder.exe" : "oporder";
const binaryPath = path.join(__dirname, binName);

if (!fs.existsSync(binaryPath)) {
  console.error("oporder: the platform binary wasn't downloaded during install.");
  console.error("Try reinstalling: npm install -g oporder");
  console.error("Or build from source: https://github.com/codetocloudorg/oporder");
  process.exit(1);
}

const result = spawnSync(binaryPath, process.argv.slice(2), { stdio: "inherit" });
if (result.error) {
  console.error("oporder: failed to run the downloaded binary:", result.error.message);
  process.exit(1);
}
process.exit(result.status === null ? 1 : result.status);
