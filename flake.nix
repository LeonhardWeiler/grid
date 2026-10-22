{
  description = "grid - Go realtime canvas";
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };
  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
        frontend = pkgs.buildNpmPackage {
          pname = "grid-frontend";
          version = "0.1.0";
          src = ./.;
          nodejs = pkgs.nodejs_24;
          npmDepsHash = "sha256-D/5CgpG1WDxFGM9idFuYDCIkGjDqy1QiCzZ4R7nDxRo=";
          preBuild = "npm test && npm exec tsc -- --noEmit";
          installPhase = "cp -r public/dist $out";
        };
      in {
        packages.default = pkgs.buildGoModule {
          pname = "grid";
          version = "0.1.0";
          src = ./.;
          vendorHash = "sha256-7JqwcenKhaOgziXfkl32qz2VrZ0zKhNJMLfvHto/Pco=";
          preCheck = ''
            test -z "$(gofmt -l main.go internal)"
            go vet ./...
          '';
          checkFlags = [ "-race" ];
          postInstall = ''
            mkdir -p $out/share
            ln -s ${frontend} $out/share/grid
          '';
        };
        devShells.default = pkgs.mkShell {
          packages = with pkgs; [ go gopls gotools air git bun nodejs_24 ];
        };
      });
}
