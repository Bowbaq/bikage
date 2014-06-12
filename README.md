bikage
======

Approximate personal milage on a citi bike, since the dawn of time

Intall
------

```bash
go get github.com/Bowbaq/bikage
```

Usage
-----

```bash
# Username / password needed to retrieve list of trips
# Google API key (with Directions API enabled) needed to 
# compute biking distance between stations
bikage -u <citike username> -p <citibike password> -k <google API key>
```
