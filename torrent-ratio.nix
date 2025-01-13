{ lib, buildGo121Module }:

# go 1.22 + zig 0.11 gives `cgo error: unsupported linker arg: -x`
buildGo121Module {
  name = "torrent-ratio";
  src =
    with lib.fileset;
    toSource {
      root = ./.;
      fileset = difference ./. (unions [
        (maybeMissing ./result)
        ./.github
        ./flake.lock
        ./flake.nix
      ]);
    };
  vendorHash = "sha256-yDaALsAg+j9gQOTx4kdeCDE85talRsbbXzo/btdryYc=";
  ldflags = [
    "-s"
    "-w"
  ];
}
