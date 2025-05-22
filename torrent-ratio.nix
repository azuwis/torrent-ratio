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
  vendorHash = "sha256-sCZA12XM6yzgmHMGn1HUnEz4EPTIMsBJdbM8RszmQPo=";
  ldflags = [
    "-s"
    "-w"
  ];
}
