{ config, pkgs, ... }:

let
  user = "pi";
  password = "pi";
in
{
  imports = [
    ../../modules/nixos/meterplus-to-victoriametrics.nix
  ];

  nixpkgs.hostPlatform = "aarch64-linux";

  # Decrypt on the Pi so rebuilds can materialize runtime-only secret files
  # without checking any plaintext into the repo.
  sops = {
    age.sshKeyPaths = [ "/etc/ssh/ssh_host_ed25519_key" ];

    secrets = {
      SWITCHBOT_TOKEN = {
        sopsFile = ../../../secrets/meterplus.yaml;
        format = "yaml";
      };

      SWITCHBOT_CLIENT_SECRET = {
        sopsFile = ../../../secrets/meterplus.yaml;
        format = "yaml";
      };

      SWITCHBOT_METERPLUS_DEVICE_ID = {
        sopsFile = ../../../secrets/meterplus.yaml;
        format = "yaml";
      };

    };

    templates."meterplus-to-victoriametrics.env" = {
      owner = user;
      group = user;
      content = ''
        SWITCHBOT_TOKEN=${config.sops.placeholder.SWITCHBOT_TOKEN}
        SWITCHBOT_CLIENT_SECRET=${config.sops.placeholder.SWITCHBOT_CLIENT_SECRET}
        SWITCHBOT_METERPLUS_DEVICE_ID=${config.sops.placeholder.SWITCHBOT_METERPLUS_DEVICE_ID}
      '';
    };
  };

  nix.gc = {
    automatic = true;
    dates = "daily";
    # Keep the Pi's storage from filling up with old generations between rebuilds.
    options = "--delete-older-than 7d";
  };

  boot = {
    # kernelPackages = pkgs.linuxKernel.packages.linux_rpi4;
    initrd.availableKernelModules = [ "xhci_pci" "usbhid" "usb_storage" ];
    loader = {
      grub.enable = false;
      generic-extlinux-compatible.enable = true;
    };
  };

  fileSystems = {
    "/" = {
      device = "/dev/disk/by-label/NIXOS_SD";
      fsType = "ext4";
      options = [ "noatime" ];
    };
  };

  networking = {
    hostName = "rpi4-01";
    # Use NetworkManager so WiFi recovery and reprovisioning can be done
    # interactively on the device instead of baking credentials into the image.
    networkmanager.enable = true;
  };

  # Match the deployed site so logs and scheduled jobs line up with operators.
  time.timeZone = "Asia/Tokyo";
  i18n.defaultLocale = "en_US.UTF-8";

  environment.systemPackages = with pkgs; [ vim ];

  services.openssh = {
    enable = true;
    settings = {
      AuthenticationMethods = "publickey";
      KbdInteractiveAuthentication = false;
      PubkeyAuthentication = true;
    };
  };

  services.prometheus.exporters.node = {
    enable = true;
    # Keep host metrics private because VictoriaMetrics scrapes them locally.
    listenAddress = "127.0.0.1";
  };

  services.victoriametrics = {
    enable = true;
    prometheusConfig.scrape_configs = [
      {
        job_name = "node";
        # Scrape the local exporter directly so host metrics land in the same
        # database as the SwitchBot metrics without adding another agent.
        static_configs = [
          {
            targets = [ "127.0.0.1:9100" ];
          }
        ];
      }
    ];
  };

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
        # Keep dashboards declarative so rebuilds recreate the same baseline UI.
        options.path = ./dashboards;
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

  users = {
    # Allow the first-boot password in the image to be rotated in place.
    mutableUsers = true;
    groups.pi = { };
    users."${user}" = {
      isNormalUser = true;
      group = user;
      # Let the operator repair WiFi locally with nmtui without editing Nix first.
      extraGroups = [ "wheel" "networkmanager" ];
      initialPassword = password;
      openssh.authorizedKeys.keys = [
        "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIIsL/08gzz5N0JmhfyeBTOSUG1ObAeYS99u39ScgW3Oj"
      ];
    };
  };

  security.sudo.wheelNeedsPassword = false;

  networking.firewall.allowedTCPPorts = [
    3000
    8428
  ];

  hardware.enableRedistributableFirmware = true;
  # New images should use the same release defaults as the distributed base OS.
  system.stateVersion = "26.11";
}
