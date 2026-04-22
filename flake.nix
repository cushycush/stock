{
  description = "stock — package/tool/runtime installer, companion to store";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        # Bump alongside every tagged release; Nix auto-derives the version
        # string passed to `stock --version` from this.
        version = "0.2.0";
      in
      {
        packages = {
          default = self.packages.${system}.stock;

          stock = pkgs.buildGoModule {
            pname = "stock";
            inherit version;

            src = ./.;

            # Replace with the hash printed on first `nix build`. Until the
            # tree is actually built under nix we don't have a reliable way to
            # precompute this, so it ships as a fakeHash sentinel.
            vendorHash = "sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=";

            ldflags = [
              "-s"
              "-w"
              "-X main.version=v${version}"
            ];

            subPackages = [ "cmd/stock" ];

            meta = with pkgs.lib; {
              description = "Package/tool/runtime installer, companion to store";
              homepage = "https://github.com/cushycush/stock";
              license = licenses.mit;
              mainProgram = "stock";
              platforms = platforms.unix ++ platforms.windows;
            };
          };
        };

        apps.default = flake-utils.lib.mkApp {
          drv = self.packages.${system}.stock;
        };

        devShells.default = pkgs.mkShell {
          # Mirror the toolchain CI uses so local iteration matches release.
          packages = [
            pkgs.go_1_26
            pkgs.gopls
            pkgs.gotools
          ];
        };
      });
}
