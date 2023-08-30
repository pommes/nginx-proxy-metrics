package geoip

import (
	"github.com/oschwald/geoip2-golang"
	"log"
	"net"
)

type GeoLite2Locator struct{}

func (locator GeoLite2Locator) Lookup(ip string) IPLocation {
	db, err := geoip2.Open("GeoLite2-City.mmdb")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	parsedIp := net.ParseIP(ip)

	record, err := db.City(parsedIp)
	if err != nil {
		log.Fatal(err)
	}

	ipLocation := IPLocation{}

	ipLocation.Country = record.Country.Names["en"]
	ipLocation.City = record.City.Names["en"]

	return ipLocation
}