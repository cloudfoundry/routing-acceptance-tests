# Routing Acceptance Tests

This test suite exercises [Cloud Foundry Routing](https://github.com/cloudfoundry/routing-release) deployment.

> **Note**: This repository should be imported as `code.cloudfoundry.org/routing-acceptance-tests`.

## Running test suites

### Test setup

To run the Routing Acceptance tests or Smoke tests, you will need:
- a running deployment of [routing-release](https://github.com/cloudfoundry/routing-release)
- the latest version of the [rtr CLI](https://github.com/cloudfoundry/routing-api-cli/releases)
- an environment variable `CONFIG` which points to a `.json` file that contains the router api endpoint
```bash
git clone https://github.com/cloudfoundry/routing-release.git
cd routing-release
./scripts/update
source .envrc
```

The following commands will create a config file `integration_config.json` for a
[bosh-lite](https://bosh.io/docs/quick-start/) installation and set the
`CONFIG` environment variable to the path for this file. Edit
`integration_config.json` as appropriate for your environment.

### Running Acceptance tests

```bash
cd ~/workspace/routing-release/src/code.cloudfoundry.org/routing-acceptance-tests/
cat > integration_config.json <<EOF
{
  "addresses": ["10.244.14.2"],
  "api": "api.bosh-lite.com",
  "admin_user": "admin",
  "admin_password": "admin",
  "skip_ssl_validation": true,
  "use_http":true,
  "apps_domain": "bosh-lite.com",
  "include_http_routes": true,
  "default_timeout": 120,
  "cf_push_timeout": 120,
  "verbose": true,
  "test_password": "test",
  "oauth": {
    "token_endpoint": "https://uaa.bosh-lite.com",
    "client_name": "tcp_emitter",
    "client_secret": "tcp-emitter-secret",
    "port": 443,
    "skip_ssl_validation": true
  },
  "tcp_router_group": "default-tcp"
}
EOF
export CONFIG=$PWD/integration_config.json
./bin/test

```


### Running Smoke tests

```bash
cd ~/workspace/routing-release/src/code.cloudfoundry.org/routing-acceptance-tests/
cat > integration_config.json <<EOF
{
  "addresses": ["10.244.14.2"],
  "api": "api.bosh-lite.com",
  "admin_user": "admin",
  "admin_password": "admin",
  "skip_ssl_validation": true,
  "use_http":true,
  "default_timeout": 120,
  "apps_domain": "bosh-lite.com",
  "tcp_apps_domain": "tcp.bosh-lite.com",
  "oauth": {
    "token_endpoint": "https://uaa.bosh-lite.com",
    "client_name": "tcp_emitter",
    "client_secret": "tcp-emitter-secret",
    "port": 443,
    "skip_ssl_validation": true
  },
  "tcp_router_group": "default-tcp"
}
EOF
export CONFIG=$PWD/integration_config.json
./bin/smoke_tests

```

### Description of Config Fields
- `addresses` - contains the IP addresses of the TCP Routers and/or the Load Balancer's IP address. IP `10.24.14.2` is IP address of `tcp_router_z1/0` job in routing-release. If this IP address happens to be different in your deployment then change the entry accordingly. The `addresses` property also accepts DNS entry for tcp router, e.g. `tcp.bosh-lite.com`.
- `admin_user` and `admin_password` - refers to the admin user used to perform a CF login with the cf CLI.
- `skip_ssl_validation` - used for the cf CLI when targeting an environment.
- `include_http_routes` (optional) - a boolean used to run tests for the experimental HTTP routing endpoints of the Routing API.
- `verbose` (optional) - a boolean which allows for the `-v` flag to be passed when running the router acceptance tests errand
- `test_password` (optional) -  By default, users created during the routing acceptance tests are configured with a random name and password. If manually configured, this property enables specifying the password for the user created during the test. `test_password` performs the same function as the manifest property, `user_password`.
- `tcp_router_group` - The router group to use for creating tcp routes.
- If `tcp_apps_domain` property is empty, smoke tests create a temporary shared domain and use the `addresses` field to connect to TCP application.
- Optionally run the smoke tests in verbose mode: `./bin/smoke_tests -v`.
- `tcp_router_group` - The router group to use for creating tcp routes.

## Development

### <a name="dependencies"></a>Dependencies

This repository's dependencies are managed using
[routing-release](https://github.com/cloudfoundry/routing-release). Please refer to documentation in that repository for setting up tests

### Executables

1. `bin/test.bash`: This file is used to run test in Docker & CI. Please refer to [Dependencies](#dependencies) for setting up tests.

### Reporting issues and requesting features

Please report all issues and feature requests in [cloudfoundry/routing-release](https://github.com/cloudfoundry/routing-release).
