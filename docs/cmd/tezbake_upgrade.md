docs/cmd/tezbake_upgrade.md## tezbake upgrade

Upgrades BB.

### Synopsis

Upgrades BB instance.

```
tezbake upgrade [flags]
```

### Options

```
      --dal               Upgrade dal.
  -h, --help              help for upgrade
      --node              Upgrade node.
      --pay               Upgrade pay.
      --peak              Upgrade peak.
      --signer            Upgrade signer.
      --skip-ami-setup    Skip ami upgrade
  -s, --upgrade-storage   Upgrade storage during the upgrade.
```

### Options inherited from parent commands

```
  -l, --log-level string       Sets output log format (json/text/auto) (default "info")
  -o, --output-format string   Sets output log format (json/text/auto) (default "auto")
  -p, --path string            Path to bake buddy instance (default "/bake-buddy")
      --version                Prints tezbake version
```

### SEE ALSO

* [tezbake](/tezbake/reference/cmd/tezbake)	 - tezbake CLI

###### Auto generated by spf13/cobra on 21-Jun-2025
