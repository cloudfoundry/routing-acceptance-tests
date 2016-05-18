# Routing Acceptance Tests

This test suite exercises [Cloud Foundry Routing](https://github.com/cloudfoundry-incubator/routing-release) deployment.

## Running Acceptance tests

### Test setup

To run the Routing Acceptance tests, you will need:
- a running routing-release deployment
- the latest version of the [rtr CLI](https://github.com/cloudfoundry-incubator/routing-api-cli/releases)
- an environment variable `CONFIG` which points to a `.json` file that contains the router api endpoint
- environment variable GOPATH set to root directory of [routing-release](https://github.com/cloudfoundry-incubator/routing-release)
```bash
git clone https://github.com/cloudfoundry-incubator/routing-release.git
cd routing-release
./scripts/update
source .envrc
```

The following commands will create a config file `integration_config.json` for a [bosh-lite](https://github.com/cloudfoundry/bosh-lite) installation and set the `CONFIG` environment variable to the path for this file. Edit `integration_config.json` as appropriate for your environment.


```bash
cd ~/workspace/routing-release/src/github.com/cloudfoundry-incubator/cf-routing-acceptance-tests/
cat > integration_config.json <<EOF
{
  "addresses": ["10.244.14.2"],
  "api": "api.bosh-lite.com",
  "admin_user": "admin",
  "admin_password": "admin",
  "skip_ssl_validation": true,
  "use_http":true,
  "apps_domain": "bosh-lite.com",
  "oauth": {
    "token_endpoint": "https://uaa.bosh-lite.com",
    "client_name": "tcp_emitter",
    "client_secret": "tcp-emitter-secret",
    "port": 443,
    "skip_oauth_tls_verification": true
  }
}
EOF
export CONFIG=$PWD/integration_config.json
```

Note:
- The `addresses` property contains the IP addresses of the TCP Routers and/or the Load Balancer's IP address. IP `10.24.14.2` is IP address of `tcp_router_z1/0` job in routing-release. If this IP address happens to be different in your deployment then change the entry accordingly.
- `admin_user` and `admin_password` properties refer to the admin user used to perform a CF login with the cf CLI.
- `skip_ssl_validation` is used for the cf CLI when targeting an environment.

### Running the tests

After correctly setting the `CONFIG` environment variable, the following command will run the tests:

```
    ./bin/test
```
