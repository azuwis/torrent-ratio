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
  vendorHash = "sha256-bEinJCUNZcUewrilaU3dJ8he3fgFmqYrToyLrBIen80=";
  ldflags = [
    "-s"
    "-w"
  ];
}
