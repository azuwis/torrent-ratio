{ lib, buildGoModule }:

buildGoModule {
  pname = "torrent-ratio";
  version = "0.8";
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
  vendorHash = "sha256-RH38kK6r357K9YgjgyHxd9iSlBK7i1MgwETN9NNeVU4=";
  ldflags = [
    "-s"
    "-w"
  ];
}
