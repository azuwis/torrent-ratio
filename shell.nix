{ }:

let
  sources = import ./sources.nix { };
  pkgs = import sources.nixpkgs { };
  devshell = import sources.devshell { nixpkgs = pkgs; };
in

devshell.mkShell {
  packages = with pkgs; [
    gcc
    gnumake
    go
    nix-update
    sqlite
  ];
  commands = [
    {
      name = "update";
      command = "nix-update torrent-ratio --version=skip";
    }
  ];
}
