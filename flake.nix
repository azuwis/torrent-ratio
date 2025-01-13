{
  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-24.05";
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
          mkCrossPackage =
            gosystem:
            let
              GOARCH = builtins.elemAt (builtins.split "_" gosystem) 2;
              GOOS = builtins.elemAt (builtins.split "_" gosystem) 0;
              ZIGARCH =
                {
                  arm64 = "aarch64";
                  amd64 = "x86_64";
                }
                .${GOARCH};
              ZIGOS = { darwin = "macos"; }.${GOOS} or GOOS;
              lib = pkgs.lib;
              zigExtraArgs =
                let
                  inherit (inputs.nixpkgs.legacyPackages.aarch64-darwin.darwin) apple_sdk;
                  inherit (apple_sdk) Libsystem;
                  inherit (apple_sdk.frameworks) CoreFoundation Security;
                in
                lib.optionalString (GOOS == "darwin")
                  " -isystem ${Libsystem}/include -F${CoreFoundation}/Library/Frameworks -F${Security}/Library/Frameworks";
            in
            torrent-ratio.overrideAttrs (
              old:
              {
                inherit GOARCH GOOS;
                # https://github.com/ziglang/zig/issues/20689
                # error: unable to create compilation: AccessDenied
                nativeBuildInputs = old.nativeBuildInputs ++ [ pkgs.zig_0_11 ];
                preBuild =
                  (old.preBuild or "")
                  + ''
                    export XDG_CACHE_HOME="$TMPDIR"
                    export CC="zig cc -target ${ZIGARCH}-${ZIGOS}${zigExtraArgs}"
                    export CXX="$CC"
                  '';
                postInstall =
                  (old.postInstall or "")
                  + ''
                    if [ -d $out/bin/${gosystem} ]
                    then
                      mv $out/bin/${gosystem}/* $out/bin
                      rmdir $out/bin/${gosystem}/
                    fi
                  '';
              }
              // lib.optionalAttrs (gosystem == "darwin_amd64") {
                # https://github.com/ziglang/zig/issues/15438
                NIX_HARDENING_ENABLE = "pie";
              }
            );
        in
        eachGoSystem mkCrossPackage // { default = torrent-ratio; }
      );

      devShells = eachSystem (
        { pkgs, devshell, ... }:
        {
          default = devshell.mkShell {
            packages = with pkgs; [
              go_1_21
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
