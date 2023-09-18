# libdns-sitehost

This is a SiteHost provider for the [libdns](https://github.com/libdns/libdns) project. It provides 
functionality to manage DNS records on SiteHost. This can be used for projects like 
[certmagic](https://github.com/caddyserver/certmagic) for DNS based challenges.

## Installation

```
go get -u github.com/sitehostnz/libdns-sitehost
```

## Usage

```go
package main

import (
	"os"

	"github.com/libdns/libdns"
	sitehost "github.com/sitehostnz/libdns-sitehost"
)

func main() {
	// Set up certmagic to use DNS challenge/response
	certmagic.DefaultACME.DNS01Solver = &certmagic.DNS01Solver{
		DNSProvider: &sitehost.Provider{
			ClientID: os.Getenv("SITEHOST_CLIENT_ID"),
			APIKey:   os.Getenv("SITEHOST_API_KEY"),
		},
	}

	// Start serving HTTP/HTTPS
	err := certmagic.HTTPS([]string{"yourdomain.co.nz"}, HTTP_HANDLER)
	if err != nil {
		panic(err)
	}
}
```

