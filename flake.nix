{
  description = "green";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs";
    blueprint = {
      url = "github:numtide/blueprint";
      inputs.nixpkgs.follows = "nixpkgs";
    };
    git-hooks = {
      url = "github:cachix/git-hooks.nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
    flake-utils.url = "github:numtide/flake-utils";
    sops-nix = {
      url = "github:Mic92/sops-nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };


  outputs = inputs:
    let
      blueprintOutputs = inputs.blueprint {
        inherit inputs;

        # Keep the project-level layout under nix/ so app code and host code do
        # not get mixed together as the Raspberry Pi setup grows.
        prefix = "nix/";
      };
    in
    blueprintOutputs
    // {
      nixosConfigurations.rpi4-01 = inputs.nixpkgs.lib.nixosSystem {
        system = "aarch64-linux";
        modules = [
          inputs.sops-nix.nixosModules.sops
          ./nix/hosts/rpi4-01/configuration.nix
        ];
      };
    };
}
