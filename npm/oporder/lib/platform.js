// Maps Node's process.platform/process.arch onto GoReleaser's naming
// (see .goreleaser.yaml at the repo root) for the six real binaries
// v0.1.0 actually published. Returns null for anything not built.
function target() {
  const goos = { darwin: "darwin", linux: "linux", win32: "windows" }[process.platform];
  const goarch = { x64: "amd64", arm64: "arm64" }[process.arch];
  if (!goos || !goarch) return null;
  return { goos, goarch };
}

function assetName(goos, goarch, version) {
  const ext = goos === "windows" ? "zip" : "tar.gz";
  return `oporder_${version}_${goos}_${goarch}.${ext}`;
}

function binaryName(goos) {
  return goos === "windows" ? "oporder.exe" : "oporder";
}

module.exports = { target, assetName, binaryName };
