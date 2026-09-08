// Shared by postinstall.js and bin/oporder.js so the exact same honest
// message reaches a user whether they see it during `npm install` or by
// actually trying to run the `oporder` command afterward — added
// because a real `npm install -g oporder` run showed this package
// silently installs nothing and prints nothing by default (current npm
// suppresses postinstall output unless run with --foreground-scripts),
// leaving a user with no `oporder` command and no explanation either.
module.exports = function printMessage() {
  console.log("");
  console.log("OpOrder has no released binary yet.");
  console.log("This package reserves the name on npm — it does not install a working tool.");
  console.log("");
  console.log("Follow progress:  https://github.com/codetocloudorg/oporder");
  console.log("Read the spec:    https://github.com/codetocloudorg/oporder/blob/main/SPEC.md");
  console.log("Site:             https://oporder.dev");
  console.log("");
  console.log("Try the reasoning engine now (needs Go, no binary required):");
  console.log("  git clone https://github.com/codetocloudorg/oporder && cd oporder && go run ./cmd/sample-report");
  console.log("");
};
