# dependency-track-exporter

A re-implementation of the unmaintained jetstack/dependency-track-exporter to export Prometheus metrics from [Dependency Track](https://dependencytrack.org/).

Whilst I have tried to maintain a similar set of command line arguments and output there is no code shared between this and the original.

## Usage

* `--help` Show context-sensitive help (also try `--help-long` and `--help-man`).
* `--web.listen-address=9916` Address to listen on for metrics.
* `--web.metrics-path="/metrics"` Path under which to expose metrics.
* `--dtrack.address=DTRACK.ADDRESS` Dependency-Track server address (default: `http://localhost:8080` or `$DEPENDENCY_TRACK_ADDR`)
* `--dtrack.api-key=DTRACK.API-KEY` Dependency-Track API key (default: `$DEPENDENCY_TRACK_API_KEY`)
* `--log.level=info` Only log messages with the given severity or above. One of: `debug`, `info`, `warn`, `error`
* `--log.format=logfmt` Output format of log messages. One of: `logfmt`, `json`
* `--poll.period=1800s` Frequency of requests to Dependency-Track.
* `--client.request-timeout=10s` Timeout value for client requests to Dependency Track.
* `--version` Show application version.

The API key currently only requires the VIEW_PORTFOLIO permission.

## Architecture

This exporter is a single binary that runs in a container, it polls the Dependency Track API for data every <poll.period> and makes that
same data available to all Prometheus scrapes until the next poll.

In my environment Dependency Track data updates a lot less frequently than standard Prometheus scrapes, and gathering this information can be expensive.

Despite this caching of the data, it is still recommended that you configure a dedicated scraper on Prometheus with a more suitable scrape interval than 15s.



## Metrics

This implementation currently supports only three metrics, a big reduction from the original.
The primary reason for this is that this is all I need.

If you want more metrics please feel free to submit a PR, or an issue.

* `dependency_track_projects_total` Total number of projects in the Dependency Track instance.
* `dependency_track_components_total` Total number of components in the Dependency Track instance.
* `dependency_track_vulnerabilities_total` Total number of vulnerabilities in the Dependency Track instance.
