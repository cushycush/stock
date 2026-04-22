# dogfood fixture

Lives here to back `make dogfood` at the repo root. The top-level
Makefile mounts this directory into a throwaway container at
`/root/dotfiles`, drops you into an interactive shell, and lets you
run `stock` commands against real package managers in a clean-slate
environment.

Not something end users should copy — it's scaffolding for catching
shell-out bugs that unit tests can't reach.

## What's in here

```text
hack/dogfood/
  .store/
    packages.yaml      # mixed-distro core group + linux-only dev group
    hooks/
      pre-install      # logs STORE_* env vars to .store/hook-env.log
      post-install     # appends to .store/hook-log
```

Running `stock install` inside the container writes:

- `.store/hook-env.log` — the `STORE_*` env the hook saw. Cross-check
  against [store-core's hook env contract](https://github.com/cushycush/store-core#hook-env-contract).
- `.store/hook-log` — two lines, one per phase, with timestamps and
  the `STORE_ACTION` value.

Both files stay inside the mounted dir so you can inspect them from
the host after exiting the container.

## Intentional quirks

- The `ghost` group references a fake manager key (`my-typo`). `stock
  doctor` should flag it as unservable and `stock install` should emit
  an unknown-manager warning. If either stays silent, file a bug.
- Alt package names across managers (`fd` on pacman/brew vs `fd-find`
  on apt/dnf) exercise the per-manager lookup path — don't "fix" them
  to match.
