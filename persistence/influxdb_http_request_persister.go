package persistence

import (
	"fmt"
	"github.com/influxdata/influxdb1-client/v2"
	"log"
	"nginx-proxy-metrics/config"
	"nginx-proxy-metrics/logline"
	"strconv"
)

const SeriesName = "http_requests"

type InfluxdbHttpRequestPersister struct {
	influxClient client.Client
}

func (persister *InfluxdbHttpRequestPersister) Setup() {
	dbClient, err := client.NewHTTPClient(client.HTTPConfig{
		Addr: config.InfluxUrl,
	})

	if err != nil {
		log.Fatal("Could not setup persistence client, reason: ", err)
	}

	_, dbErr := queryDB(dbClient, fmt.Sprintf("CREATE DATABASE %s", config.InfluxDbName))
	if dbErr != nil {
		log.Fatal("Could not create database, reason: ", dbErr)
	}

	log.Println(fmt.Sprintf("  - Altering retention policy %s_retention to %s", config.InfluxDbName, config.InfluxDbRetentionDuration))
	_, dbErr = queryDB(dbClient, fmt.Sprintf("ALTER RETENTION POLICY %s_retention ON %s DURATION %s DEFAULT",
		config.InfluxDbName, config.InfluxDbName, config.InfluxDbRetentionDuration))
	if dbErr != nil {
		log.Println("  - Could not ALTER retention policy, reason: ", dbErr)

		log.Println(fmt.Sprintf("  - Creating retention policy %s_retention with %s", config.InfluxDbName, config.InfluxDbRetentionDuration))
		_, dbErr = queryDB(dbClient, fmt.Sprintf("CREATE RETENTION POLICY %s_retention ON %s DURATION %s REPLICATION 1 DEFAULT",
			config.InfluxDbName, config.InfluxDbName, config.InfluxDbRetentionDuration))
		if dbErr != nil {
			log.Println("  - Could not CREATE retention policy, reason: ", dbErr)
		}
	}

	persister.influxClient = dbClient
}

func (persister InfluxdbHttpRequestPersister) Persist(httpRequest logline.HttpRequest) {
	batchPoints, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database: config.InfluxDbName,
	})
	if err != nil {
		log.Fatal(err)
	}

	tags := map[string]string{
		"host":            httpRequest.Host,
		"request_method":  httpRequest.RequestMethod,
		"http_version":    httpRequest.HttpVersion,
		"http_status":     strconv.Itoa(httpRequest.HttpStatus),
		"browser":         httpRequest.Browser,
		"browser_version": httpRequest.BrowserVersion,
		"os":              httpRequest.Os,
		"mobile":          strconv.FormatBool(httpRequest.Mobile),
		"country":         httpRequest.Country,
		"city":            httpRequest.City,
		"instance":        config.InfluxDbTagInstance,
		"bot":             strconv.FormatBool(httpRequest.Bot),
	}

	fields := map[string]interface{}{
		"source_ip":       httpRequest.SourceIp,
		"request_path":    httpRequest.RequestPath,
		"body_bytes_sent": httpRequest.BodyBytesSent,
		"http_referer":    httpRequest.HttpReferer,
		"user_agent":      httpRequest.UserAgent,
		"latency":         httpRequest.Latency,
	}

	point, err := client.NewPoint(SeriesName, tags, fields, httpRequest.Timestamp)
	if err != nil {
		log.Fatal(err)
	}

	batchPoints.AddPoint(point)

	if err := persister.influxClient.Write(batchPoints); err != nil {
		log.Println("Could not insert into persistence, reason:", err)
		return
	}
}

func queryDB(clnt client.Client, cmd string) (res []client.Result, err error) {
	q := client.Query{
		Command: cmd,
	}
	if response, err := clnt.Query(q); err == nil {
		if response.Error() != nil {
			return res, response.Error()
		}
		res = response.Results
	} else {
		return res, err
	}
	return res, nil
}