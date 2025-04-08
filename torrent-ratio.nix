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
  vendorHash = "sha256-acFh8eiqBisHu5MQTW2u/yPujemWbGA57lC/OniXIus=";
  ldflags = [
    "-s"
    "-w"
  ];
}
