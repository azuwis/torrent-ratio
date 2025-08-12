{ lib, buildGoModule }:

buildGoModule {
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
  vendorHash = "sha256-oG2+BiHRsDMrGdi4Apw3i04WT4lKbMH/ErY7XOpnXNc=";
  ldflags = [
    "-s"
    "-w"
  ];
}
