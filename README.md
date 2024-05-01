# btshow
BitTorrent Tracker cli and lib client in Go

### Implemented:
* [BEP15](https://www.bittorrent.org/beps/bep_0015.html)
    * UDP IPv4
        * connect
        * scrape


### Missing:
* announce
* IPv6 support
* HTTP trackers ([BEP23](https://www.bittorrent.org/beps/bep_0023.html))

## Usage
### Building from source
```bash
git clone https://github.com/shyba/btshow.git
cd btshow
go build .
```

### Testing
```bash
go test ./...
```

### Usage
### Cli
```bash
./btshow scrape <infohash> <infohash>
```

Example:
```
$ ./btshow scrape 81af07491915415dad45f87c0c2ae52fae92c06b 2aa4f5a7e209e54b32803d43670971c4c8caaa05
81af07491915415dad45f87c0c2ae52fae92c06b
Completed: 0
Leechers: 121
Seeders: 44
2aa4f5a7e209e54b32803d43670971c4c8caaa05
Completed: 0
Leechers: 235
Seeders: 61
```