package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	queryTemplate = template.New("query").Funcs(template.FuncMap{
		// Now takes a duration in Go's time format and returns an RFC3339-formatted
		// timestamp of the current time +/- the duration.
		"Now": func(t string) (string, error) {
			d, err := time.ParseDuration(t)
			if err != nil {
				return "", err
			}
			return time.Now().UTC().Add(d).Format(time.RFC3339), nil
		},
	})
)

func queryHandler(w http.ResponseWriter, r *http.Request) error {
	// Calling r.ParseForm to populate r.Form.
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("queryHandler: error while running r.ParseForm(): %s", err)
	}

	// Extracting the GraphQL API endpoint relative to the exporter.
	// It is required to provide an endpoint. If multiple endpoints are provided,
	// only the first value will be considered.
	endpoint := r.FormValue("endpoint")
	if endpoint == "" {
		return errors.New("queryHandler: no querying endpoint provided")
	}

	// Extracting the GraphQL queries to perform against the endpoint.
	// It is required to provide at least one query. If multiple queries are
	// provided, there must not be intersection between the data returned by them.
	queries, ok := r.Form["query"]
	if !ok {
		return errors.New("queryHandler: no queries provided")
	}

	// Performing template value substitution on the queries.
	var queryBuffer bytes.Buffer
	for i, query := range queries {
		t, err := queryTemplate.Parse(query)
		if err != nil {
			return fmt.Errorf("queryHandler: error while template-parsing query %d: %s", i, err)
		}

		err = t.Execute(&queryBuffer, nil)
		if err != nil {
			return fmt.Errorf("queryHandler: error while templating query %d: %s", i, err)
		}
		queries[i] = queryBuffer.String()
		queryBuffer.Reset()
	}

	// Passing all other query string parameters as query variables.
	variables := make(map[string]interface{})
	for key, values := range r.Form {
		if key != "endpoint" && key != "query" {
			variables[key] = values[len(values)-1]
		}
	}

	// Authenticating with the GraphQL endpoint is done by passing the scrape HTTP
	// header over to the endpoint verbatim, so:
	//  - if you need Basic Auth, query graphql_exporter with Basic Auth; or
	//  - if you need OAuth 2.0, query graphql_exporter with a bearer token; etc.
	header := r.Header
	delete(header, "Accept")
	delete(header, "Accept-Encoding")

	// Instantiating Client.
	client, err := NewWithHeader(endpoint, &header)
	if err != nil {
		return fmt.Errorf("queryHandler: error while instantiating Client: %s", err)
	}

	// Instantiating prometheus.Registry.
	registry := prometheus.NewRegistry()
	gauges := map[string]*prometheus.GaugeVec{}

	// Making the requests concurrently.
	requests := make([]*Request, len(queries))
	for i, query := range queries {
		requests[i] = &Request{Query: query, Variables: variables}
	}
	responsesChannel, preQueryingErrorsChannel := client.QueryMany(requests)

	var postQueryingErrors []interface{}

	// Parsing query responses as they come in.
	// Additionally, this is how we wait for Client.QueryMany's goroutines
	// to finish, as that will be notified by closing the responsesChannel and
	// preQueryingErrorsChannel channels.
	for response := range responsesChannel {
		// Appending query errors to a global error slice.
		// This is so that all queries get to be evaluated before returning, which
		// allows returning all errors together so that query debugging is not an
		// iterative process where you fix query 1, then scrape again, fix query 2,
		// scrape again, fix query N... You get to see all errors the first time.
		postQueryingErrors = append(postQueryingErrors, response.Errors...)

		// Skipping parsing if there were errors already.
		if len(postQueryingErrors) == 0 {
			// Parsing query response data as metrics.
			responseDataAsMetrics(registry, gauges, prometheus.Labels{"endpoint": endpoint}, []string{"query"}, &response.Data)
		}
	}

	// Evaluate incoming pre-query errors.
	// In the event of a pre-querying error, only the first error to be sent to
	// preQueryingErrorsChannel will be returned, as opposed to post-query errors.
	for error := range preQueryingErrorsChannel {
		return fmt.Errorf("queryHandler: pre-querying errors: %s", error)
	}

	// Return global error slice if any queries returned errors.
	// This implies that if query N had errors, yet queries N-1 and N+1 did not,
	// no metrics will be returned for any of those queries since the response
	// body will be used to send the errors back to the client.
	if len(postQueryingErrors) > 0 {
		postQueryingErrorsJSON, _ := json.Marshal(postQueryingErrors)
		return fmt.Errorf("queryHandler: post-querying errors: %s", postQueryingErrorsJSON)
	}

	// Writing the metrics into the scrape response.
	promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(w, r)
	return nil
}

func responseDataAsMetrics(r *prometheus.Registry, g map[string]*prometheus.GaugeVec, labels prometheus.Labels, path []string, data *interface{}) {
	// Recursively inspect data and register its values as metrics by the
	// following convention, given that all metric values must be float64s:

	switch (*data).(type) {

	// Boolean values are represented as a 1 if true, 0 otherwise.
	case bool:
		if (*data).(bool) {
			setGaugeValue(r, g, labels, path, 1)
		} else {
			setGaugeValue(r, g, labels, path, 0)
		}

	// Number values are converted into float64 without changing their value.
	case float64:
		setGaugeValue(r, g, labels, path, (*data).(float64))

	// String values are not supported by Prometheus, so:
	//   - the string value is stored as the value of a "value" label; and
	//   - if the string value is representable as a 64-bit floating point number,
	//     it is converted and stored as the gauge value of the metric;
	//   - if not, the gauge value is set to 1.
	// Beware, however, that if a value changed since its last scrape, it will not
	// be returned as 0, since we don't know about its existence now.
	case string:
		labels["value"] = (*data).(string)
		if value, err := strconv.ParseFloat((*data).(string), 64); err == nil {
			setGaugeValue(r, g, labels, path, value)
		} else {
			setGaugeValue(r, g, labels, path, 1)
		}
		delete(labels, "value")

	// Arrays are recursively inspected.
	// To uniquely identify items within arrays, a label is added to all metrics
	// contained in the array. Its name is the last value in path, and its value
	// is the index of the item within the array (0..n).
	case []interface{}:
		for i, value := range (*data).([]interface{}) {
			labels[path[len(path)-1]] = strconv.Itoa(i)
			responseDataAsMetrics(r, g, labels, path, &value)
		}
		delete(labels, path[len(path)-1])

	// Objects as recursively inspected.
	// Objects and nested objects have their keys appended to the metric name, so
	// for example, if a query's data is:
	//   {"com":{"example":{"www":3.14}}}
	// graphql_exporter would return:
	//   query_com_example_www{endpoint="..."} 3.14
	// As you may have guessed, this implies all object keys must be composed
	// exclusively by characters allowed within metric names in Prometheus'
	// Exposition format. More info:
	//   https://prometheus.io/docs/concepts/data_model/#metric-names-and-labels
	case map[string]interface{}:
		for key, value := range (*data).(map[string]interface{}) {
			responseDataAsMetrics(r, g, labels, append(path, key), &value)
		}

	}
}

func setGaugeValue(r *prometheus.Registry, g map[string]*prometheus.GaugeVec, labels prometheus.Labels, path []string, value float64) {
	var labelNames []string
	for key := range labels {
		labelNames = append(labelNames, key)
	}
	sort.Strings(labelNames)

	name := strings.Join(path, "_")
	key := strings.Join(append([]string{name}, labelNames...), ",")

	gauge, ok := g[key]
	if !ok {
		gauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: name}, labelNames)
		g[key] = gauge
		r.Register(gauge)
	}
	gauge.With(labels).Set(value)
}
