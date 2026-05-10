{ lib, buildGoModule }:

buildGoModule (finalAttrs: {
  pname = "torrent-ratio";
  version = "0.11";
  src =
    with lib.fileset;
    toSource {
      root = ./.;
      fileset = unions [
        ./go.mod
        ./go.sum
        ./main.go
        ./main_test.go
        ./static
        ./templates
      ];
    };
  vendorHash = "sha256-QasbiVTEOY2Zr/MaZ2EeX/69vMhlSmoP86MbLcuE8xI=";
  ldflags = [
    "-X main.Version=v${finalAttrs.version}"
    "-s"
  ];
})
