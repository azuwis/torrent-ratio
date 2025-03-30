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
  vendorHash = "sha256-R2Xja18MHwuuyiJXJK2gCNFZrFxySDDbKg8onNxHR2I=";
  ldflags = [
    "-s"
    "-w"
  ];
}
