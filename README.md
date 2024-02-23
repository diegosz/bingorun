# bingorun

> This is a **Production Draft** ¯\\_(ツ)_/¯ , it has no tests, no CI, no docs,
> no examples, etc. It's just a proof of concept whit good intentions and charm.

Tool for running [`bingo`](https://github.com/bwplotka/bingo) managed tools.

## Requirements

- Go 1.18+
- [`bingo`](https://github.com/bwplotka/bingo)

## Install

```shell
go install github.com/diegosz/bingorun@latest
```

Recommended: Pin the `bingorun` tool to your project using `bingo`. Do it via:

```shell
bingo get -l github.com/diegosz/bingorun@latest
```

## Usage

Run you desired tool using the following command:

```shell
bingorun <tool-name> [args...]
```

It runs the specified tool, and (re)installs the tool if missing.

Example:

```shell
bingorun go-enum --marshal --nocase -f=<file.go>
```

It could be used in `go:generate` directives, for example:

```go
//go:generate bingorun go-enum --marshal --nocase -f=$GOFILE
```

Instead of the tool name, you can use the following commands:

```shell
    -b, --bin       print the path of the tool binary
    -v, --version   print the version
    -h, --help      print this help message
```

## Motivation

I like to use [bingo](https://github.com/bwplotka/bingo) to automatically manage
the versioning of Go package level binaries required as dev tools for a project.

It's an awesome tool that even allows you to use some versioned commands from
bigger projects like `Kubernetes`, `Prometheus` or `Thanos` without the use of a
lot of `replace` statements (plus others like `exclude` or `retract`).

Also I like to keep the `go:generate` directives in the source code, close to
the place that it belongs instead of having it far away in a makefile or some
other script. For example:

```go
//go:generate go-enum --marshal --nocase -f=$GOFILE
```

When using `go:generate` it also means that we will have defined 3 environment
variables with the name of the **go package**, the **filename** where the
directive is located and also the exact **line** the generate command was
invoked from.

- `$GOFILE`:    The base name of the file.
- `$GOLINE`:    The line number of the directive in the source file.
- `$GOPACKAGE`: The name of the package of the file containing the directive.

We could install the tools using the `bingo get -l` symlink option to solve this
case, but when jumping from one project to the other, we could have side effects
by having another binary with same name or different version.

I completely agree with the notion of not including a `run` command within
`bingo`, as mentioned
[here](https://github.com/bwplotka/bingo/issues/52#issuecomment-751444495) and
[here](https://github.com/bwplotka/bingo/issues/98), but no one mentioned
anything about not having a `run` command outside `bingo`. :-)

## TODO

- [ ] Add tests.
- [ ] Add documentation.
- [ ] Add examples.
- [ ] Add continuous integration.

## Credits

- [bingo](https://github.com/bwplotka/bingo) by
  [@bwplotka](https://bwplotka.dev), many thanks!
