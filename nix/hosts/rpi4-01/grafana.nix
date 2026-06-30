{ ... }:

{
  services.grafana = {
    enable = true;
    provision.datasources.settings = {
      datasources = [
        {
          name = "VictoriaMetrics";
          type = "prometheus";
          uid = "victoriametrics";
          isDefault = true;
          # Query over loopback so dashboards stay usable even if LAN routing changes.
          url = "http://127.0.0.1:8428";
          editable = false;
        }
      ];
    };
    provision.dashboards.settings.providers = [
      {
        name = "default";
        # Keep dashboards next to this host's Grafana config so host-owned UI
        # changes stay local without introducing a shared abstraction too early.
        options.path = ./grafana/dashboards;
      }
    ];
    settings = {
      # Keep the key stable across rebuilds so Grafana can read its own state
      # without forcing a separate secret distribution path for this host.
      security.secret_key = "SW2YcwTIb9zpOOhoPsMm";
      server = {
        # Bind on the LAN so the dashboard stays usable from other devices.
        http_addr = "0.0.0.0";
        http_port = 3000;
      };
    };
  };
}
