package main

import (
  "log"
  "github.com/fsouza/go-dockerclient"
  "io"
  "time"
  "bufio"
)

const DOCKER_DAEMON_SOCKET = "unix:///var/run/docker.sock"

type DockerContainerLogMiner struct {
  logPersistor *ILogPersistor
  logParser *ILogParser
}

func (dockerContainerLogMiner *DockerContainerLogMiner) SetLogPersistor(logPersistor ILogPersistor) {
  dockerContainerLogMiner.logPersistor = &logPersistor
}

func (dockerContainerLogMiner *DockerContainerLogMiner) SetLogParser(logParser ILogParser) {
  dockerContainerLogMiner.logParser = &logParser
}

func (dockerContainerLogMiner *DockerContainerLogMiner) ParseAndPersistStdPipesOutput(stdout, stderr io.Reader) {
  listenToPipe := func(input io.Reader) {
    buf := bufio.NewReader(input)

    for {
      line, _ := buf.ReadString('\n')


      parsedLogLine, err := (*dockerContainerLogMiner.logParser).Parse(line)

      if err != nil {
        continue
      }

      (*dockerContainerLogMiner.logPersistor).Persist(parsedLogLine)
    }
  }

  log.Println("Listening to stdout and stderr pipes")

  go listenToPipe(stdout)
  go listenToPipe(stderr)
}


func (dockerContainerLogMiner *DockerContainerLogMiner) Mine() {
  containerId := findProxyContainerId()

  log.Println("Attaching container log listener.")

  client, err := docker.NewClient(DOCKER_DAEMON_SOCKET)
  if err != nil {
    panic(err)
  }

  sinceTime := time.Now()

  stdoutReader, stdoutWriter := io.Pipe()
  stderrReader, stderrWriter := io.Pipe()

  dockerContainerLogMiner.ParseAndPersistStdPipesOutput(stdoutReader, stderrReader)

  log.Println("Starting to get logs from docker daemon.")
  for {
    dockerLogErr := client.Logs(docker.LogsOptions{
      Container:         containerId,
      OutputStream:      stdoutWriter,
      ErrorStream:       stderrWriter,
      Stdout:            true,
      Stderr:            true,
      Follow:            true,
      Tail:              "all",
      Since:             sinceTime.Unix(),
      InactivityTimeout: 0,
    })

    sinceTime = time.Now()

    if dockerLogErr != nil {
      panic(dockerLogErr)
    }
  }
}

func findProxyContainerId() string {
  log.Println("Finding proxy container.")

  client, err := docker.NewClient(DOCKER_DAEMON_SOCKET)
  if err != nil {
    panic(err)
  }

  proxyContainerName := getProxyContainerName()

  filters := make(map[string][]string)
  filters["name"] = append(filters["name"], proxyContainerName)

  log.Println("Getting proxy container with name: ", proxyContainerName)
  containers, err := client.ListContainers(docker.ListContainersOptions{Filters: filters})
  if err != nil {
    panic(err)
  }

  if len(containers) <= 0 {
    panic("No running container found with specified name.")
  }

  proxyContainer := containers[0]
  proxyContainerId := proxyContainer.ID

  return proxyContainerId
}

func getProxyContainerName() string {
  return GetEnvOrPanic(PROXY_CONTAINER_NAME_ENV_NAME)
}