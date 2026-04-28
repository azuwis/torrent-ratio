{ lib, buildGoModule }:

buildGoModule (finalAttrs: {
  pname = "torrent-ratio";
  version = "0.10";
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
  vendorHash = "sha256-+eqtiSDnAXlFqjOVunEDD862gf4k569/DLpmshQBJog=";
  ldflags = [
    "-X main.Version=v${finalAttrs.version}"
    "-s"
  ];
})
