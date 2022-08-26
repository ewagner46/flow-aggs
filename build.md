# Requirements

This was tested on OSX 12.4. It requires
```
go
```
I used go 1.19 but earlier versions will also likely work.

# Build 
From the root directory
```
./build.sh
```

This install all library dependencies, build `flowaggs-server`, and copy it to
the top-level directory, where it can be run as

`./flowaggs-server`

See `config.yaml` for annotated configuration options and the `README.md`
for additional context.
