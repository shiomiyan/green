{
  inputs,
  pkgs,
  system,
  ...
}:

let
  git-hooks = inputs.git-hooks.lib.${system};
  formatter = import ../formatter.nix {
    inherit inputs pkgs;
  };
  preCommitCheck = git-hooks.run {
    src = ../..;
    hooks.betterleaks = {
      enable = true;
      name = "Detect hardcoded secrets";
      description = "Detect hardcoded secrets using Betterleaks";
      entry = "betterleaks git --pre-commit --redact --staged --no-banner";
      language = "system";
      pass_filenames = false;
      stages = [ "pre-commit" ];
      package = pkgs.betterleaks;
    };
    hooks.treefmt = {
      enable = true;
      package = formatter;
    };
  };
in
pkgs.mkShell {
  packages =
    (with pkgs; [
      # Keep the shell focused on the tools we use locally so editing and
      # formatting stay predictable during day-to-day host work.
      go
      gcc
      nixd
    ])
    ++ preCommitCheck.enabledPackages;

  shellHook = preCommitCheck.shellHook;
}
