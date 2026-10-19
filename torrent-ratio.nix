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
  vendorHash = "sha256-j4PNw7tJD2Ovy7CZXVKDsoFqD4voBYqr3W8bDAe3mMQ=";
  ldflags = [
    "-X main.Version=v${finalAttrs.version}"
    "-s"
  ];
})
