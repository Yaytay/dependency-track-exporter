# dependency-track-exporter

A re-implementation of the unmaintained jetstack/dependency-track-exporter.

## Flags

* `--help` Show context-sensitive help (also try `--help-long` and `--help-man`).
* `--web.listen-address=9916` Address to listen on for metrics.
* `--web.metrics-path="/metrics"` Path under which to expose metrics.
* `--dtrack.address=DTRACK.ADDRESS` Dependency-Track server address (default: `http://localhost:8080` or `$DEPENDENCY_TRACK_ADDR`)
* `--dtrack.api-key=DTRACK.API-KEY` Dependency-Track API key (default: `$DEPENDENCY_TRACK_API_KEY`)
* `--log.level=info` Only log messages with the given severity or above. One of: `debug`, `info`, `warn`, `error`
* `--log.format=logfmt` Output format of log messages. One of: `logfmt`, `json`
* `--poll.period=1800s` Frequency of requests to Dependency-Track.
* `--client-request-timeout=10s` Timeout value for client requests to Dependency Track.
* `--version` Show application version.