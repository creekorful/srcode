# srcode

Tool that help developers to manage their codebase in an effective & productive way.

## How to install

### Using homebrew

srcode is available as a Brew tap:

```
$ brew tap creekorful/srcode
$ brew install srcode
```

## How to clone an existing codebase

```
$ srcode clone git@github.com:creekorful/dot-srcode.git ~/Projects
```

this will clone the codebase located at remote and clone it to `~/Projects`.

## How to initialize a new codebase

```
$ srcode init --remote git@github.com:creekorful/dot-srcode.git ~/Projects
```

this will initialize a new codebase with given remote, at given directory.

## Create & use custom script

You can create custom script in your codebase:

```
$ srcode script test go test -v ./...
```

And then run the script directly like this:

```
$ srcode run test
```

or simpler:

```
$ srcode test
```