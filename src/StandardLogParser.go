package main

import (
  "regexp"
  "strconv"
  "errors"
)

type StandardLogParser struct {
}

const LOG_LINE_REGEX = `^(\S+) *\|\s+(\S+)\s+(\S+).+\[(.+)\]\s+"([^"]+)"\s+(\S+)\s+(\S+)\s+"([^"]+)"\s+"([^"]+)"`

func (standardLogParser StandardLogParser) Parse(logLine string) (ParsedLogLine, error) {
  var logLineParserRegex = regexp.MustCompile(LOG_LINE_REGEX)

  logLineParserRegexResult := logLineParserRegex.FindStringSubmatch(logLine)
  if len(logLineParserRegexResult) <= 0 {
    return ParsedLogLine{}, errors.New("Log line did not match nginx log line.")
  }

  containerName := logLineParserRegexResult[1]

  if !isProxyContainer(containerName) {
    return ParsedLogLine{}, errors.New("Container name does not match proxy container name.")
  }

  host := logLineParserRegexResult[2]
  remoteAddress := logLineParserRegexResult[3]
  timestamp := logLineParserRegexResult[4]
  httpRequest := logLineParserRegexResult[5]

  var httpRequestRegex = regexp.MustCompile(`^(\S+)\s+(\S+)\s+(\S+)`)
  httpRequestRegexResult := httpRequestRegex.FindStringSubmatch(httpRequest)
  requestType := httpRequestRegexResult[1]
  requestPath := httpRequestRegexResult[2]
  httpVersion := httpRequestRegexResult[3]

  httpStatus, _ := strconv.Atoi(logLineParserRegexResult[6])
  bodyBytesSent, _ := strconv.Atoi(logLineParserRegexResult[7])
  httpReferer := logLineParserRegexResult[8]
  userAgent := logLineParserRegexResult[9]

  parsedLogLine := ParsedLogLine{}
  parsedLogLine.containerName = containerName
  parsedLogLine.host = host
  parsedLogLine.sourceIp = remoteAddress
  parsedLogLine.timestamp = timestamp
  parsedLogLine.requestMethod = requestType
  parsedLogLine.requestPath = requestPath
  parsedLogLine.httpVersion = httpVersion
  parsedLogLine.httpStatus = httpStatus
  parsedLogLine.bodyBytesSent = bodyBytesSent
  parsedLogLine.httpReferer = httpReferer
  parsedLogLine.userAgent = userAgent

  parseUserAgentAndSetFields(userAgent, &parsedLogLine)
  lookupIpAndSetFields(remoteAddress, &parsedLogLine)

  return parsedLogLine, nil
}

func parseUserAgentAndSetFields(userAgentString string, parsedLogLine *ParsedLogLine) {
  mssolaUserAgentParser := MssolaUserAgentParser{}
  userAgent := mssolaUserAgentParser.Parse(userAgentString)

  parsedLogLine.browser = userAgent.browser
  parsedLogLine.browserVersion = userAgent.browserVersion
  parsedLogLine.os = userAgent.os
  parsedLogLine.mobile = userAgent.mobile
}

func lookupIpAndSetFields(ip string, parsedLogLine *ParsedLogLine) {
  geoIp2IPLookupService := GeoIp2IPLookupService{}

  ipLocation := geoIp2IPLookupService.Lookup(ip)

  parsedLogLine.country = ipLocation.country
  parsedLogLine.city = ipLocation.city
}

func isProxyContainer(containerName string) bool {
  return containerName == getProxyContainerName()
}