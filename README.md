<div align="center">
  <p><img src="https://em-content.zobj.net/thumbs/160/apple/325/fire_1f525.png" width="100px"></p>
  <h1>graphql_exporter</h1>
  <p>Prometheus exporter for <a href="https://www.graphql.com/">GraphQL</a></p>
  <code>docker pull quay.io/ricardbejarano/graphql_exporter</code>
</div>


## Usage

### Using the official container image

#### Docker Hub

Available on Docker Hub as [`docker.io/ricardbejarano/graphql_exporter`](https://hub.docker.com/r/ricardbejarano/graphql_exporter):

- [`v1.2.1`, `latest` *(Dockerfile)*](Dockerfile)

#### RedHat Quay

Available on RedHat Quay as [`quay.io/ricardbejarano/graphql_exporter`](https://quay.io/repository/ricardbejarano/graphql_exporter):

- [`v1.2.1`, `latest` *(Dockerfile)*](Dockerfile)

### Building the container image yourself

```bash
docker build -t graphql_exporter .
docker run -it -p 9199:9199 graphql_exporter
```

### Building the binary yourself

```bash
make
./bin/graphql_exporter
```

### Integrating with Prometheus

See Prometheus configuration [examples](examples):
* [Cloudflare Analytics API](examples/Cloudflare-Analytics-API.md)
