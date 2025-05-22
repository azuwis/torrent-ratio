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
  vendorHash = "sha256-J8CZfpGGzKiN3eEvG1DoletrcF/K9LajfWqE3MDKBD4=";
  ldflags = [
    "-s"
    "-w"
  ];
}
