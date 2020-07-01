# cogment-cli

The cogment CLI tool

## Building

```
docker build -t cogment-cli .
```

## Running

The image expects the current cogment project to be mounted at `/cogment`. Also, if you want to be able to execute commands that manipulate the host's docker (such as building or running a project), then you need
to bind your host's docker socket

On windows:
```
docker run --rm -v%cd%:/cogment -v//var/run/docker.sock://var/run/docker.sock cogment-cli run build
```

On linux (TODO: verify):
```
docker run --rm -v$(pwd):/cogment -v/var/run/docker.sock:/var/run/docker.sock cogment-cli run build
```

You may want to create an alias as a quick Quality of Life improvement:

```
alias cogment="docker run --rm -v$(pwd):/cogment -v//var/run/docker.sock://var/run/docker.sock cogment-cli"

cogment run build
```