{
  inputs.nixpkgs.url = "nixpkgs";

  outputs = { self, nixpkgs }:
    let
      eachSystem = nixpkgs.lib.genAttrs [
        "aarch64-darwin"
        "aarch64-linux"
        "x86_64-darwin"
        "x86_64-linux"
      ];
      eachGoSystem = nixpkgs.lib.genAttrs [
        "darwin_amd64"
        "darwin_arm64"
        "linux_amd64"
        "linux_arm64"
        "windows_amd64"
        "windows_arm64"
      ];
    in
    {
      packages = eachSystem (system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
          torrent-ratio = pkgs.buildGoModule {
            name = "torrent-ratio";
            src = with pkgs.lib.fileset; toSource {
              root = ./.;
              fileset = difference
                ./.
                (unions [
                  (maybeMissing ./result)
                  ./.github
                  ./flake.lock
                  ./flake.nix
                ]);
            };
            vendorHash = "sha256-4NAwh2sp1SBVniMmx6loFMN/9gbY3kfWnHV/U0TIgHg=";
            ldflags = [ "-s" "-w" ];
          };
          mkCrossPackage = gosystem:
            let
              GOARCH = builtins.elemAt (builtins.split "_" gosystem) 2;
              GOOS = builtins.elemAt (builtins.split "_" gosystem) 0;
              ZIGARCH = { arm64 = "aarch64"; amd64 = "x86_64"; }.${GOARCH};
              ZIGOS = { darwin = "macos"; }.${GOOS} or GOOS;
              lib = pkgs.lib;
              zigExtraArgs = with nixpkgs.legacyPackages.aarch64-darwin.darwin.apple_sdk; lib.optionalString (GOOS == "darwin")
                " -isystem ${Libsystem}/include -F${frameworks.CoreFoundation}/Library/Frameworks -F${frameworks.Security}/Library/Frameworks";
            in
            torrent-ratio.overrideAttrs (old: {
              inherit GOARCH GOOS;
              nativeBuildInputs = old.nativeBuildInputs ++ [ pkgs.zig ];
              preBuild = (old.preBuild or "") + ''
                export XDG_CACHE_HOME="$TMPDIR"
                export CC="zig cc -target ${ZIGARCH}-${ZIGOS}${zigExtraArgs}"
                export CXX="$CC"
              '';
              postInstall = (old.postInstall or "") + ''
                if [ -d $out/bin/${gosystem} ]
                then
                  mv $out/bin/${gosystem}/* $out/bin
                  rmdir $out/bin/${gosystem}/
                fi
              '';
            } // lib.optionalAttrs (gosystem == "darwin_amd64") {
              # https://github.com/ziglang/zig/issues/15438
              NIX_HARDENING_ENABLE = "pie";
            });
        in
        eachGoSystem mkCrossPackage // {
          default = torrent-ratio;
        }
      );

      devShells = eachSystem (system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        {
          default = pkgs.mkShell {
            buildInputs = with pkgs; [
              go
              sqlite
            ];
          };
        });
    };
}
