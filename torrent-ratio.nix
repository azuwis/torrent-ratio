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
  vendorHash = "sha256-Rq4/T2ueR9bHReVa2JdyyvPy+/uAdPI8V20WnjFt6w8=";
  ldflags = [
    "-X main.Version=v${finalAttrs.version}"
    "-s"
  ];
})
