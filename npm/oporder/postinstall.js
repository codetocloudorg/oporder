#!/usr/bin/env node
// Downloads the real platform binary from the matching GitHub Release
// (codetocloudorg/oporder@vX.Y.Z, where X.Y.Z is this package's own
// version — the two are kept in lockstep on purpose) and extracts it
// into bin/. bin/oporder.js (the registered "bin" entry) execs whatever
// lands here. A download failure never fails `npm install` itself —
// this is a convenience layer on top of the real, independently
// installable Go binary, not the source of truth for it.
const fs = require("fs");
const path = require("path");
const { execFileSync } = require("child_process");
const { target, assetName } = require("./lib/platform");

const pkg = require("./package.json");
const RELEASE_BASE = "https://github.com/codetocloudorg/oporder/releases/download";

async function main() {
  const t = target();
  const binDir = path.join(__dirname, "bin");
  fs.mkdirSync(binDir, { recursive: true });

  if (!t) {
    console.log("");
    console.log(`OpOrder has no prebuilt binary for ${process.platform}/${process.arch}.`);
    console.log("Build from source instead (needs Go):");
    console.log("  git clone https://github.com/codetocloudorg/oporder && cd oporder && go build -o oporder ./cmd/oporder");
    console.log("");
    return;
  }

  const version = pkg.version;
  const asset = assetName(t.goos, t.goarch, version);
  const url = `${RELEASE_BASE}/v${version}/${asset}`;
  const archivePath = path.join(binDir, asset);

  console.log(`Downloading oporder v${version} for ${t.goos}/${t.goarch}...`);

  let res;
  try {
    res = await fetch(url);
  } catch (err) {
    warnDownloadFailed(url, err.message);
    return;
  }
  if (!res.ok) {
    warnDownloadFailed(url, `HTTP ${res.status}`);
    return;
  }

  const buf = Buffer.from(await res.arrayBuffer());
  fs.writeFileSync(archivePath, buf);

  try {
    // Windows' bundled bsdtar (tar.exe, present since Windows 10 1803)
    // reads .zip as well as .tar.gz, so one command works on every
    // platform this package supports — no extra dependency to extract
    // an archive.
    execFileSync("tar", ["-xf", archivePath, "-C", binDir], { stdio: "inherit" });
  } finally {
    fs.rmSync(archivePath, { force: true });
  }

  if (t.goos !== "windows") {
    fs.chmodSync(path.join(binDir, "oporder"), 0o755);
  }

  console.log(`oporder v${version} installed.`);
}

function warnDownloadFailed(url, reason) {
  console.error("");
  console.error(`oporder: couldn't download the binary (${reason}):`);
  console.error(`  ${url}`);
  console.error("Try again with: npm install -g oporder");
  console.error("Or build from source: https://github.com/codetocloudorg/oporder");
  console.error("");
}

main().catch((err) => {
  console.error("oporder: postinstall failed:", err.message);
  console.error("Build from source instead: https://github.com/codetocloudorg/oporder");
});
