# cogment-cli

The cogment CLI tool

## Building

```
docker build -t cogment-cli .
```

## Running

The image expects the current cogment project to be mounted at `/cogment`.

```
docker run --rm -v$(pwd):/cogment cogment-cli cogment generate --python_dir=.
```