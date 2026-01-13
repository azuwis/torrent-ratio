{ }:

let
  sources = import ./sources.nix { };
  pkgs = import sources.nixpkgs { };
  torrent-ratio = pkgs.callPackage ./torrent-ratio.nix { };
  mkCrossPackage = pkgs.callPackage ./cross.nix {
    inherit torrent-ratio;
  };
in

torrent-ratio
// pkgs.lib.genAttrs [
  "darwin_amd64"
  "darwin_arm64"
  "linux_amd64"
  "linux_arm64"
  "windows_amd64"
  "windows_arm64"
] mkCrossPackage
// {
  default = torrent-ratio;
}
