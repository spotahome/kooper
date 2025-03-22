## [unreleased]

## [2.8.0] - 2025-03-22

- Update Kubernetes libraries for 1.32.
- Update Go version to v1.24.

## [2.7.0] - 2024-08-31

- Update Kubernetes libraries for 1.31.
- Update Go version to v1.23.

## [2.6.0] - 2024-03-31

- Update Kubernetes libraries for 1.29.
- Update Go version to v1.22.

## [2.5.0] - 2023-09-04

- Update Kubernetes libraries for 1.28.
- Change resource locks from configmaps to leases.

## [2.4.0] - 2023-07-04

- Update Kubernetes libraries for 1.27.

## [2.3.0] - 2022-12-07

- Update Kubernetes libraries for 1.25.

## [2.2.0] - 2021-08-30

- Update Kubernetes libraries for 1.24.

## [2.1.0] - 2021-10-07

- Update Kubernetes libraries for 1.22.

## [2.0.0] - 2020-07-24

NOTE: Breaking release in controllers.

- Refactor controller package.
- Refactor metrics package.
- Refactor log package.
- Remove operator concept and remove CRD initialization in favor of using only
  controllers and let the CRD initialization outside Kooper (e.g CRD yaml).
- Default resync time to 3 minutes.
- Default workers to 3.
- Disable retry handling on controllers in case of error by default.
- Remove tracing.
- Minimum Go version v1.13 (error wrapping required).
- Refactor Logger with structured logging.
- Add Logrus helper wrapper.
- Refactor to simplify the retrievers.
- Refactor metrics recorder implementation including the prometheus backend.
- Refactor internal controller queue into a decorator implementation approach.
- Remove `Delete` method from `controller.Handler` and simplify to only `Handle` method
- Add `DisableResync` flag on controller configuration to disable the resync of all resources.
- Update Kubernetes libraries for 1.22.

## [0.8.0] - 2019-12-11

- Support for Kubernetes 1.15.

## [0.7.0] - 2019-11-26

- Support for Kubernetes 1.14.

## [0.6.0] - 2019-06-01

### Added

- Support for Kubernetes 1.12.

### Removed

- Glog logger.

## [0.5.1] - 2019-01-19

### Added

- Shortnames on CRD registration.

## [0.5.0] - 2018-10-24

### Added

- Support for Kubernetes 1.11.

## [0.4.1] - 2018-10-07

### Added

- Enable subresources support on CRD registration.
- Category support on CRD registration.

## [0.4.0] - 2018-07-21

This release breaks Prometheus metrics.

### Added

- Grafana dashboard for the refactored Prometheus metrics.

### Changed

- Refactor metrics in favor of less metrics but simpler and more meaningful.

## [0.3.0] - 2018-07-02

This release breaks handler interface to allow passing a context (used to allow tracing).

### Added

- Context as first argument to handler interface to pass tracing context (Breaking change).
- Tracing through opentracing.
- Leader election for controllers and operators.
- Let customizing (using configuration) the retries of event processing errors on controllers.
- Controllers now can be created using a configuration struct.
- Add support for Kubernetes 1.10.

## [0.2.0] - 2018-02-24

This release breaks controllers constructors to allow passing a metrics recorder backend.

### Added

- Prometheus metrics backend.
- Metrics interface.
- Concurrent controller implementation.
- Controllers record metrics about queued and processed events.

### Fixed

- Fix passing a nil logger to make controllers execution break.

## [0.1.0] - 2018-02-15

### Added

- CRD client check for kubernetes apiserver (>=1.7)
- CRD ensure (waits to be present after registering a CRD)
- CRD client tooling
- multiple CRD and multiple controller operator.
- single CRD and single controller operator.
- sequential controller implementation.
- Dependencies managed by dep and vendored.

[unreleased]: https://github.com/spotahome/kooper/compare/v2.8.0...HEAD
[2.8.0]: https://github.com/spotahome/kooper/compare/v2.7.0...v2.8.0
[2.7.0]: https://github.com/spotahome/kooper/compare/v2.6.0...v2.7.0
[2.6.0]: https://github.com/spotahome/kooper/compare/v2.5.0...v2.6.0
[2.5.0]: https://github.com/spotahome/kooper/compare/v2.4.0...v2.5.0
[2.4.0]: https://github.com/spotahome/kooper/compare/v2.3.0...v2.4.0
[2.3.0]: https://github.com/spotahome/kooper/compare/v2.2.0...v2.3.0
[2.2.0]: https://github.com/spotahome/kooper/compare/v2.1.0...v2.2.0
[2.1.0]: https://github.com/spotahome/kooper/compare/v2.0.0...v2.1.0
[2.0.0]: https://github.com/spotahome/kooper/compare/v0.8.0...v2.0.0
[0.8.0]: https://github.com/spotahome/kooper/compare/v0.7.0...v0.8.0
[0.7.0]: https://github.com/spotahome/kooper/compare/v0.6.0...v0.7.0
[0.6.0]: https://github.com/spotahome/kooper/compare/v0.5.1...v0.6.0
[0.5.1]: https://github.com/spotahome/kooper/compare/v0.5.0...v0.5.1
[0.5.0]: https://github.com/spotahome/kooper/compare/v0.4.1...v0.5.0
[0.4.1]: https://github.com/spotahome/kooper/compare/v0.4.0...v0.4.1
[0.4.0]: https://github.com/spotahome/kooper/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/spotahome/kooper/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/spotahome/kooper/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/spotahome/kooper/releases/tag/v0.1.0
