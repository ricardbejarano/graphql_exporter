# Cloudflare GraphQL Analytics API

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
    bearer_token: "<YOUR CLOUDFLARE API TOKEN>"
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
      zoneTag: ["<YOUR CLOUDFLARE ZONE ID>"]  # a GraphQL query variable
    static_configs:
      - targets: ["127.0.0.1:9199"]  # graphql_exporter address:port
```

## Check your new metrics!

Once Prometheus has scraped the `graphql_exporter`, you should be able to query your brand new metrics.

If you used the above query, consider running the following PromQL query:

```
query_viewer_zones_httpRequests1hGroups_sum_requests
```
