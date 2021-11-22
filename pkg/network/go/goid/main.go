package goid

// This generates GetGoroutineIDOffset's implementation in ./goid_offset.go:
// - Use ../go-toolchains as the destination for the Go toolchains to be downloaded.
//   This gets cached in CI, since it takes time to download, unpack, and compile them.
//   Each toolchain version is around 500 MiB on disk.
// - Use ./build-out as the destination for the generated binary files.
//go:generate go run ./internal/generate_goid_lut.go --test-program ./internal/program.go --package goid --out ./goid_offset.go --min-go 1.15 --arch amd64,arm64 --max-quick-go 1.17.3 --shared-build-dir ../go-toolchains --out-dir ./build-out
