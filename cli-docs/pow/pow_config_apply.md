## pow config apply

Apply the default or provided storage config to the specified cid

### Synopsis

Apply the default or provided storage config to the specified cid

```
pow config apply [cid] [flags]
```

### Options

```
  -c, --conf string   Optional path to a file containing storage config json, falls back to stdin, uses storage profile default by default
  -h, --help          help for apply
  -o, --override      If set, override any pre-existing storage configuration for the cid
  -w, --watch         Watch the progress of the resulting job
```

### Options inherited from parent commands

```
      --serverAddress string   address of the powergate service api (default "127.0.0.1:5002")
  -t, --token string           storage profile auth token
```

### SEE ALSO

* [pow config](pow_config.md)	 - Provides commands to interact with cid storage configs

