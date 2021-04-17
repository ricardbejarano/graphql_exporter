<p align="center"><img src="https://emojipedia-us.s3.dualstack.us-west-1.amazonaws.com/thumbs/160/apple/271/axe_1fa93.png" width="120px"></p>
<h1 align="center">graphql_exporter</h1>
<p align="center"><a href="https://prometheus.io/">Prometheus</a> metrics <a href="https://prometheus.io/docs/instrumenting/exporters/">exporter</a> for <a href="https://www.graphql.com/">GraphQL</a></p>


# Description

GraphQL metrics exporter for Prometheus.

This piece of software has one mission: query GraphQL endpoints and transform their results into Prometheus metrics.

# Additions
Edited `query.go` so it attempts to convert a string to a float64. When graphql returns data it can often be in a string format
```
case string:
  // try to convert string to int
  data_float, err := strconv.ParseFloat((*data).(string), 32)
  if err == nil {
   setGaugeValue(r, labels, path, data_float)
  } else {
   labels["value"] = (*data).(string)
   setGaugeValue(r, labels, path, 1)
   delete(labels, "value")
  }
```


# Usage

See [examples](examples/README.md).

## With the prebuilt container image

Available on [Docker Hub](https://hub.docker.com) as [`docker.io/ricardbejarano/graphql_exporter`](https://hub.docker.com/r/ricardbejarano/graphql_exporter):

- [`1.0.0`, `latest` *(Dockerfile)*](Dockerfile)

Also available on [Quay](https://quay.io) as [`quay.io/ricardbejarano/graphql_exporter`](https://quay.io/repository/ricardbejarano/graphql_exporter):

- [`1.0.0`, `latest` *(Dockerfile)*](Dockerfile)

Any of both registries will do, example:

```bash
docker run -it -p 9199:9199 quay.io/ricardbejarano/graphql_exporter
```

## Building the container image from source

First clone the repository, and `cd` into it:

```bash
docker build -t graphql_exporter .
```

Now run it:

```bash
docker run -it -p 9199:9199 graphql_exporter
```

## Building the binary from source

First clone the repository, and `cd` into it.

```bash
make
```

Now run it:

```bash
./bin/graphql_exporter
```


# License

MIT licensed, see [LICENSE](LICENSE) for more details.
