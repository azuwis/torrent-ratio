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
  vendorHash = "sha256-ffwFjxn1UhSnwfw37cExp1c4ipfBEysao7jvH2K1cT8=";
  ldflags = [
    "-X main.Version=v${finalAttrs.version}"
    "-s"
  ];
})
