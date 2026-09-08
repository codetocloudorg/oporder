# oporder

Vendor-neutral assessment CLI for cloud migration and application modernization — reads your
code and your live infrastructure, tells you honestly what's actually there, and hands you a
plan without an opinion about which cloud you should end up on.

```
npm install -g oporder
oporder scan
```

This package downloads the real, prebuilt `oporder` binary for your platform from
[GitHub Releases](https://github.com/codetocloudorg/oporder/releases) during install — macOS
and Linux, Intel and Apple Silicon/ARM64, are supported today. The Go binary is the real
distribution; this package is a convenience wrapper around it, not a separate implementation.

- **Repository**: https://github.com/codetocloudorg/oporder
- **Specification**: https://github.com/codetocloudorg/oporder/blob/main/SPEC.md
- **Site**: https://oporder.dev

If your platform isn't one of the prebuilt targets, `npm install` says so and points you at
building from source (needs [Go](https://go.dev/dl/), nothing else):

```
git clone https://github.com/codetocloudorg/oporder && cd oporder && go build -o oporder ./cmd/oporder
```
