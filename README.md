# green

おうち監視システムです。

- OS: Raspberry PI 4 model B
- Dashboard: Grafana
- Database: VictoriaMetrics
- Misc: Switchbot

## Deploy

Pi 側で `sops-nix` が復号するため、事前に次を満たしておきます。

- `secrets/` 配下の secret を `sops` で暗号化してある
- `secrets/meterplus.yaml` の placeholder を実値に更新してある
- `.sops.yaml` の Pi recipient が実機の SSH host key 由来 recipient と一致している
- Pi 側に `/etc/ssh/ssh_host_ed25519_key` がある
- host 設定で `pi` user/group が両方宣言されている

この deploy で Go バイナリのビルド、Pi への配置、systemd timer/service の更新までを Nix に一本化しています。手作業の unit 配布を残さない理由は、実行バイナリと unit 定義のずれを防ぐためです。

`sops-nix` は activation 中に template の owner/group を解決するため、`pi` の group が無いと `setupSecrets` で止まります。先に user/group を Nix で揃えておく理由はこの順序依存を deploy 時に持ち込まないためです。

まず `nixos-rebuild` に進む前に、同じ agent socket で素の SSH が通ることを確認します。ここを分ける理由は、flake の評価失敗と SSH 認証失敗を同じコマンドに混ぜると切り分けが遅くなるためです。

```
ssh -o IdentityAgent="$SSH_AUTH_SOCK" pi@192.168.10.13 true
```

`nix-copy-closure` や remote build も同じ socket を使うように固定します。`ssh-add -L` が成功していても別 socket を見に行くと同じ公開鍵が提示されないためです。

```
export NIX_SSHOPTS="-o IdentityAgent=$SSH_AUTH_SOCK"
```

```
nixos-rebuild switch --flake "path:$PWD#rpi4-01" --target-host pi@192.168.10.13 --build-host pi@192.168.10.13 --elevate=sudo --ask-elevate-password
```
