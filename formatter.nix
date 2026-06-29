{ pkgs, ... }:

let
  configFile = (pkgs.formats.toml { }).generate ".treefmt.toml" {
    formatter.gofmt = {
      command = "gofmt";
      includes = [ "**/*.go" ];
    };

    formatter.nixfmt = {
      command = "nixfmt";
      includes = [ "**/*.nix" ];
    };
  };
in
pkgs.writeShellApplication {
  name = "treefmt";
  runtimeInputs = with pkgs; [
    treefmt
    go
    nixfmt
  ];
  text = ''
    exec ${pkgs.treefmt}/bin/treefmt --config-file ${configFile} "$@"
  '';
}
