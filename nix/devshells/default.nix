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
      # Pin the repo config explicitly so pre-commit keeps honoring the local
      # allowlist even if the hook runs from a different working directory.
      entry = "betterleaks git --config .betterleaks.toml --pre-commit --redact --staged --no-banner -v";
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
