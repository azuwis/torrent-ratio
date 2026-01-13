{ }:

let
  sources = import ./sources.nix { };
  pkgs = import sources.nixpkgs { };
  devshell = import sources.devshell { nixpkgs = pkgs; };
in

devshell.mkShell {
  packages = with pkgs; [
    go
    sqlite
  ];
}
