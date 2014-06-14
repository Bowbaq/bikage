bikage
======

Approximate personal distance covered on a citi bike

Intall
------

```bash
go get github.com/Bowbaq/bikage
```

Usage
-----
You can use https://bikage.herokuapp.com/ or if you're more of a command line person, see below:

```bash
# Username / password needed to retrieve list of trips
# Google API key (with Directions API enabled) needed to
# compute biking distance between stations
bikage -u <citike username> -p <citibike password> -k <google API key>
```
