{
  description = "Gonzo - Go-based TUI log analysis tool";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  inputs.flake-utils.url = "github:numtide/flake-utils";

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
      in {
        packages.default = pkgs.buildGoModule rec {
          pname = "gonzo";
          version = "0.1.5";
          src = ./.;

          # First run with a fake hash; Nix will print the correct vendorHash.
          vendorHash = "sha256-XKwtq8EF774lHLHtyFzveFa5agJa15CvhsuwwaQdJwU";

          # If you split binaries later, enable: subPackages = [ "cmd/gonzo" ];
          ldflags = [ "-s" "-w" ];

          meta = with pkgs.lib; {
            description = "Go-based TUI for log analysis";
            homepage = "https://github.com/control-theory/gonzo";
            license = licenses.mit;
            mainProgram = "gonzo";
            platforms = platforms.unix;
          };
        };

        apps.default = {
          type = "app";
          program = "${self.packages.${system}.default}/bin/gonzo";
        };

        devShells.default = pkgs.mkShell { buildInputs = [ pkgs.go pkgs.git ]; };
      });
}