{
  inputs.flake-utils.url = "github:numtide/flake-utils";

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        packages.default = pkgs.buildGoModule {
          name = "torrent-ratio";
          src = ./.;
          vendorHash = "sha256-4NAwh2sp1SBVniMmx6loFMN/9gbY3kfWnHV/U0TIgHg=";
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
