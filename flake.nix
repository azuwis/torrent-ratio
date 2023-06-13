{
  inputs.flake-utils.url = "github:numtide/flake-utils";

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system: let
      pkgs = nixpkgs.legacyPackages.${system};
    in {
      packages.default = pkgs.buildGoModule {
        name = "torrent-ratio";
        src = ./.;
        vendorHash = "sha256-HH0VHleShuv91QkV1CC8thgBWe5RgoUKhXa706Ked04=";
        buildInputs = [ pkgs.sqlite ];
      };
      devShells.default = pkgs.mkShell {
        buildInputs = with pkgs; [
          go
          sqlite
        ];
      };
    });
}
