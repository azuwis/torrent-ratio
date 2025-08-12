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
  vendorHash = "sha256-ZXdvTyaMUIPoSgbT8mfsdWn7U1UGAHOGOfn9PDOk2kY=";
  ldflags = [
    "-s"
    "-w"
  ];
}
