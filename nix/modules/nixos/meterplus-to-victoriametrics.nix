{ config, pkgs, ... }:

let
  package = pkgs.callPackage ../../packages/meterplus-to-victoriametrics { };
in
{
  systemd.services.meterplus-to-victoriametrics = {
    description = "Send Meter Plus metrics to VictoriaMetrics";
    after = [ "network-online.target" ];
    wants = [ "network-online.target" ];
    serviceConfig = {
      Type = "oneshot";
      EnvironmentFile = config.sops.templates."meterplus-to-victoriametrics.env".path;
      ExecStart = "${package}/bin/meterplus-to-victoriametrics";
      User = "pi";
      Group = "pi";
    };
  };

  systemd.timers.meterplus-to-victoriametrics = {
    description = "Run meterplus-to-victoriametrics every minute";
    wantedBy = [ "timers.target" ];
    timerConfig = {
      OnCalendar = "*-*-* *:*:00";
      AccuracySec = "1s";
      Persistent = true;
      Unit = "meterplus-to-victoriametrics.service";
    };
  };
}
