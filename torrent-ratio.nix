{ lib, buildGo121Module }:

# go 1.22 + zig 0.11 gives `cgo error: unsupported linker arg: -x`
buildGo121Module {
  name = "torrent-ratio";
  src =
    with lib.fileset;
    toSource {
      root = ./.;
      fileset = unions [
        ./go.mod
        ./go.sum
        ./main.go
        ./static
        ./templates
      ];
    };
  vendorHash = "sha256-bEinJCUNZcUewrilaU3dJ8he3fgFmqYrToyLrBIen80=";
  ldflags = [
    "-s"
    "-w"
  ];
}
