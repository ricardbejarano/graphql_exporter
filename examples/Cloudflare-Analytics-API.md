# Cloudflare Analytics API

## Run `graphql_exporter`

Nothing special, simply run `graphql_exporter`:

```bash
./bin/graphql_exporter
```

## Configure Prometheus

Insert the following into your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: "cloudflare-analytics"
    metrics_path: "/query"
    bearer_token: "<YOUR_CLOUDFLARE_API_TOKEN>"
    params:
      endpoint: ["https://api.cloudflare.com/client/v4/graphql"]
      query:
        - |
          {
            viewer {
              zones(filter: {zoneTag: $zoneTag}) {
                httpRequests1hGroups(limit: 1, filter: {datetime_geq: "{{ Now "-1h" }}"}) {
                  dimensions {
                    datetime
                  }
                  sum {
                    bytes
                    cachedBytes
                    cachedRequests
                    requests
                  }
                }
              }
            }
          }
      zoneTag: ["<YOUR_CLOUDFLARE_ZONE_ID>"]  # a GraphQL query variable
    static_configs:
      - targets: ["127.0.0.1:9199"]  # graphql_exporter address:port
```
