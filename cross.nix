{
  lib,
  apple_sdk,
  torrent-ratio,
  zig_0_11,
}:
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
  zigExtraArgs =
    let
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
    nativeBuildInputs = old.nativeBuildInputs ++ [ zig_0_11 ];
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
)
