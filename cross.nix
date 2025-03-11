{
  torrent-ratio,
  zig,
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
in

torrent-ratio.overrideAttrs (old: {
  env = {
    inherit GOARCH GOOS;
    # When building .#darwin_arm64 with `CGO_ENABLED=1`, error: unable to find dynamic system library 'resolv'
    CGO_ENABLED = 0;
  };
  nativeBuildInputs = old.nativeBuildInputs ++ [
    zig
  ];
  preBuild =
    (old.preBuild or "")
    + ''
      export CC="zig cc -target ${ZIGARCH}-${ZIGOS}"
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
})
