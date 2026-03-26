{ lib, buildGoModule }:

buildGoModule (finalAttrs: {
  pname = "torrent-ratio";
  version = "0.9";
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
  vendorHash = "sha256-pGmwukdTmQAeaNrce2F+hNGRHMR04kKO+VKE75azVF4=";
  ldflags = [
    "-X main.Version=v${finalAttrs.version}"
    "-s"
  ];
})
