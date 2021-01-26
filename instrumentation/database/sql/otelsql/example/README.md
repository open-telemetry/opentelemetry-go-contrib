# database/sql instrumentation example

A MySQL client using database/sql with instrumentation.

These instructions expect you have
[docker-compose](https://docs.docker.com/compose/) installed.

Bring up the `Mysql` services to run the
example:

```sh
docker-compose up -d mysql
```

Then up the `client` service to make request with `MySQL`:

```sh
docker-compose up client
```

Shut down the services when you are finished with the example:

```sh
docker-compose down
```
