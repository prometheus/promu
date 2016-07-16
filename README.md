# Prometheus Utility Tool [![Build Status](https://travis-ci.org/prometheus/promu.svg)][travis]

[![CircleCI](https://circleci.com/gh/prometheus/promu/tree/master.svg?style=shield)][circleci]

## Usage

```
promu is the utility tool for Prometheus projects

Usage:
  promu [flags]
  promu [command]

Available Commands:
  build       Build a Go project
  crossbuild  Crossbuild a Go project using Golang builder Docker images
  info        Print info about current project and exit
  release     Upload tarballs to the Github release
  tarball     Create a tarball from the builded Go project
  version     Print the version and exit

Flags:
      --config string   Config file (default is ./.promu.yml)
  -v, --verbose         Verbose output
      --viper           Use Viper for configuration (default true)

Use "promu [command] --help" for more information about a command.
```

## `.promu.yml` config file

See documentation example [here](doc/examples/.promu.yml)

## More information

  * This tool is part of our reflexion about [Prometheus component Builds](https://docs.google.com/document/d/1Ql-f_aThl-2eB5v3QdKV_zgBdetLLbdxxChpy-TnWSE)
  * All of the core developers are accessible via the [Prometheus Developers Mailinglist](https://groups.google.com/forum/?fromgroups#!forum/prometheus-developers) and the `#prometheus` channel on `irc.freenode.net`.

## Contributing

Refer to [CONTRIBUTING.md](CONTRIBUTING.md)

## License

Apache License 2.0, see [LICENSE](LICENSE).


[circleci]: https://circleci.com/gh/prometheus/promu
[travis]: https://travis-ci.org/prometheus/promu

