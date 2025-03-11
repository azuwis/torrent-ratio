{
  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
  inputs.devshell.url = "github:numtide/devshell";
  inputs.devshell.inputs.nixpkgs.follows = "nixpkgs";

  outputs =
    inputs@{ ... }:
    let
      systems = [
        "aarch64-darwin"
        "aarch64-linux"
        "x86_64-darwin"
        "x86_64-linux"
      ];
      eachSystem =
        f:
        inputs.nixpkgs.lib.genAttrs systems (
          system:
          f rec {
            inherit system;
            pkgs = inputs.nixpkgs.legacyPackages.${system};
            devshell = import inputs.devshell { nixpkgs = pkgs; };
          }
        );
      eachGoSystem = inputs.nixpkgs.lib.genAttrs [
        "darwin_amd64"
        "darwin_arm64"
        "linux_amd64"
        "linux_arm64"
        "windows_amd64"
        "windows_arm64"
      ];
    in
    {
      packages = eachSystem (
        { pkgs, ... }:
        let
          torrent-ratio = pkgs.callPackage ./torrent-ratio.nix { };
          mkCrossPackage = pkgs.callPackage ./cross.nix {
            inherit torrent-ratio;
          };
        in
        eachGoSystem mkCrossPackage // { default = torrent-ratio; }
      );

      devShells = eachSystem (
        { pkgs, devshell, ... }:
        {
          default = devshell.mkShell {
            packages = with pkgs; [
              go
              sqlite
            ];
          };
        }
      );

      apps = eachSystem (
        { pkgs, ... }:
        {
          update = {
            type = "app";
            program = builtins.toString (
              pkgs.writers.writeBash "update" ''
                ${pkgs.nix-update}/bin/nix-update -F default --version=skip --override-filename torrent-ratio.nix
              ''
            );
          };
        }
      );
    };
}
