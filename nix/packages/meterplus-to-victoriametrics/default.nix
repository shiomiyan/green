{ pkgs, ... }:

pkgs.buildGoModule {
  pname = "meterplus-to-victoriametrics";
  version = "0.1.0";
  src = ../../..;
  subPackages = [ "cmd/meterplus-to-victoriametrics" ];
  vendorHash = null;
}
