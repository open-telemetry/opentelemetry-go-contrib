## Integration Example

### To run the example:
1. `cd`  into the example directory.
2. Run `docker-compose up`.
3. Wait for cassandra to listen for cql clients with the following message in the logs: 

```
Server.java:159 - Starting listening for CQL clients on /0.0.0.0:9042 (unencrypted)...
```

4. Run the example using `go run .`.

5. You can view the spans in the browser at `localhost:9411` and the metrics at `localhost:2222`.

### When you're done:
1. `ctrl+c` to stop the example program.
2. `docker-compose down` to stop cassandra and zipkin.
