# Prometheus Remote Write Exporter for Cortex Example

This example uses [Docker Compose](https://docs.docker.com/compose/) to set up:
1. A Go program that creates two instruments and exports randomly generated metrics data
   using the exporter
2. An instance of [Cortex](https://cortexmetrics.io/) to receive the metrics data
3. An instance of [Grafana](https://grafana.com/) to visualize the exported data

## Requirements

- [Docker Compose](https://docs.docker.com/compose/)

Installation instructions can be found in the Docker
[documentation](https://docs.docker.com/compose/install/).

## Instructions

1. Run the docker container with the following command:
   ```bash
   docker-compose up -d
   ```
   The `-d` flag causes all services to be run in detached mode, or in the background.
   This causes no logs to show up. Users can attach themselves to a service's logs
   manually.

2. Log in to the Grafana instance at [http://localhost:3000](http://localhost:3000)
   * The login credentials are admin/admin
   * There may be an additional screen on setting a new password. It isn't needed and can
     be skipped

3. Go to the Data Sources tab page 
   * Look for a gear icon on the left sidebar and select Data Sources

4. Add a new Prometheus Data Source
   * Use
     [http://host.docker.internal:9009/api/prom/](http://host.docker.internal:9009/api/prom/)
     as the URL
   * Optionally, set the scrape interval to 3s to make updates appear quickly
   * Click `Save & Test`
  
5. Go to the New Dashboard page
   * Look for a + sign and select Dashboard
   * Click `Add New Panel`

6. Add new metric queries
   * Click the `+ Query` button to create a new query if an empty one isn't available
   * Under the `Metrics` dropdown, select a metric
   * Add new queries to see different metrics at the same time
   * Optionally, adjust the time range by clicking the `Last 6 hours` button on the upper
     right side of the graph
   * Optionally, set up auto-refresh by selecting an option under the dropdown next to the
     refresh button on the upper right side of the graph
   * Click the refresh button and data should show up on the graph

7. Shut down the services when you're finished with the example

   ```bash
   docker-compose down
   ```