# Cortex Exporter Example

This example exports several metrics to a [Cortex](https://cortexmetrics.io/) instance and displays
them in [Grafana](https://grafana.com/).

## Requirements

- [Docker Compose](https://docs.docker.com/compose/) installed

## Instructions

1. Run the docker container with the following command

```bash
docker-compose up -d
```

2. Log in to the Grafana instance running at [http://localhost:3000](http://localhost:3000). The
   login credentials are admin/admin.

3. Add Cortex as a data source by creating a new Prometheus data source using
   [http://localhost:9009/api/prom/](http://localhost:9009/api/prom/) as the endpoint.

4. View collected metrics in Grafana.

5. Shut down the services when you're finished with the example

```bash
docker-compose down
```
