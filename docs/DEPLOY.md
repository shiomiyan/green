`nixos-rebuild`を使って、ホスト端末からraspiをプロビジョニング、デプロイします。

```
nixos-rebuild switch --flake "path:$PWD#rpi4-01" --target-host pi@192.168.10.13 --build-host pi@192.168.10.13 --elevate=sudo --ask-elevate-password
```

## Credential

[sops](https://github.com/getsops/sops) / [sops-nix](https://github.com/Mic92/sops-nix)で管理します。

## SSHができないとき

SSH Agent Forwardingがflakyなのか、raspiのSSHに失敗する場合にはforwardingし直してみる。

```
export SSH_AUTH_SOCK="$HOME/.ssh/agent.sock" && rm -f "$SSH_AUTH_SOCK" && (setsid socat UNIX-LISTEN:"$SSH_AUTH_SOCK",fork EXEC:"npiperelay.exe -ei -s //./pipe/openssh-ssh-agent",nofork >/dev/null 2>&1 &)
```
