# Pyroscope Configuration

Pyroscope is a continuous profiling platform that helps you understand your application's performance characteristics.

## What is Continuous Profiling?

Continuous profiling collects performance data from your running application:
- **CPU usage**: Which functions consume the most CPU time
- **Memory allocation**: Where your app allocates memory
- **Goroutines**: Number and state of goroutines
- **Mutex/Block contention**: Where your app waits for locks

## Accessing Pyroscope

- **URL**: http://localhost:4040
- **API**: http://localhost:4040/api

## Integration

The Go application automatically sends profiling data to Pyroscope when running in development mode.

### Profiling Types Enabled:

1. **CPU** - CPU time spent in functions
2. **Alloc Objects** - Number of allocated objects
3. **Alloc Space** - Amount of memory allocated
4. **Inuse Objects** - Live objects in memory
5. **Inuse Space** - Memory currently in use
6. **Goroutines** - Goroutine count
7. **Mutex Count** - Mutex lock contention count
8. **Mutex Duration** - Time spent waiting for mutexes
9. **Block Count** - Blocking operation count
10. **Block Duration** - Time spent in blocking operations

## Using Pyroscope with Grafana

Pyroscope data source is automatically provisioned in Grafana (http://localhost:3000):

1. Login to Grafana (admin/admin)
2. Go to Explore
3. Select "Pyroscope" data source
4. Query your application profiles

## Configuration

Environment variables (in `.env`):
```bash
PYROSCOPE_SERVER_ADDRESS=http://pyroscope:4040
```

## Useful Queries

In Grafana Explore with Pyroscope datasource:

### View CPU profile
```
go-microservice-template{env="development"}
```

### View memory allocations
```
go-microservice-template{env="development",__profile_type__="alloc_space"}
```

### Compare different time periods
Use the time range selector to compare before/after changes

## Performance Impact

- **Minimal overhead**: ~1-5% CPU overhead
- **Safe for production**: Can be enabled with sampling
- **Adjustable**: Profile types can be selectively enabled

## Troubleshooting

### Pyroscope not receiving data

1. Check app logs for connection errors
2. Verify Pyroscope is running: `docker ps | grep pyroscope`
3. Check network connectivity: `docker exec <app-container> curl http://pyroscope:4040`

### High memory usage

Reduce profiling frequency or disable some profile types in `cmd/main.go`
