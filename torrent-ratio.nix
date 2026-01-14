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
  vendorHash = "sha256-v+G1wWQwnBQRh5HZ6m5B7qb3KGY3N2NO2oibS+ISDIk=";
  ldflags = [
    "-X main.Version=v${finalAttrs.version}"
    "-s"
    "-w"
  ];
})
