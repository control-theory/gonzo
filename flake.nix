{
  description = "Gonzo - Go-based TUI log analysis tool";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  inputs.flake-utils.url = "github:numtide/flake-utils";
  inputs.gomod2nix.url = "github:nix-community/gomod2nix";
  inputs.gomod2nix.inputs.nixpkgs.follows = "nixpkgs";

  outputs = {
    self,
    nixpkgs,
    flake-utils,
    gomod2nix,
  }:
    flake-utils.lib.eachDefaultSystem (system: let
      pkgs = import nixpkgs {inherit system;};
      buildGoApplication = gomod2nix.legacyPackages.${system}.buildGoApplication;
    in {
      packages.default = buildGoApplication rec {
        pname = "gonzo";
        version = "0.1.5";
        src = ./.;
        modules = ./gomod2nix.toml;

        # If you split binaries later, enable: subPackages = [ "cmd/gonzo" ];
        ldflags = ["-s" "-w"];

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

      devShells.default = pkgs.mkShell {
        buildInputs = [
          pkgs.go
          pkgs.git
          gomod2nix.legacyPackages.${system}.gomod2nix
        ];
      };
    });
}
