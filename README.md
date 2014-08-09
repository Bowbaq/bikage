bikage
======

Approximate personal distance covered on a Citi Bike

Intall
------

```bash
# For the library
go get github.com/Bowbaq/bikage

# For the cli client
go install github.com/Bowbaq/bikage/cmd/bikage
```

A web client is available at https://bikage.herokuapp.com/

Usage
-----

```bash
-> % bikage -help
Usage of bikage:
  -google-api-key="": Google API key, directions API must be enabled (required)
  -mongo-url="": MongoDB url (persistent distance cache) (optional, defaults to local JSON cache)
  -p="": citibike.com password (required)
  -u="": citibike.com username (required)
```
