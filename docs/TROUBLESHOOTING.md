npiperelayでWindowsのSSH Agentを使う。

```
export SSH_AUTH_SOCK="$HOME/.ssh/agent.sock"
rm -f "$SSH_AUTH_SOCK"
setsid socat UNIX-LISTEN:"$SSH_AUTH_SOCK",fork EXEC:"npiperelay.exe -ei -s //./pipe/openssh-ssh-agent",nofork >/dev/null 2>&1 &
ssh-add -L
```

`ssh-add -L` と NixOS の authorized key が一致していても、SSH client が別の agent socket を見ていれば `Permission denied (publickey)` になります。まず素の SSH を同じ socket で通し、その後に `nixos-rebuild` へ進めます。

```
ssh -vvv -o IdentityAgent="$SSH_AUTH_SOCK" pi@192.168.10.13 true
```

`Offering public key` の行に期待する `ssh-ed25519` が出ない場合は、鍵不一致ではなく client 側の agent 参照がずれています。`nixos-rebuild` も同じ socket に固定します。

```
export NIX_SSHOPTS="-o IdentityAgent=$SSH_AUTH_SOCK"
```
