# Routing Acceptance Tests

This test suite exercises [Cloud Foundry Routing](https://github.com/cloudfoundry-incubator/cf-routing-release) deployment.

## Running Acceptance tests

### Test setup

To run the Routing Acceptance tests, you will need:
- a running cf-routing-release deployment
- an environment variable `CONFIG` which points to a `.json` file that contains the router api endpoint
- make sure that your GOPATH is set to root directory of [cf-routing-release](https://github.com/cloudfoundry-incubator/cf-routing-release) 
```bash
git clone https://github.com/cloudfoundry-incubator/cf-routing-release.git
cd cf-routing-release
./scripts/update
source .envrc
```

The following commands will create a config file `integration_config.json` for a [bosh-lite](https://github.com/cloudfoundry/bosh-lite) installation and set the `CONFIG` environment variable to the path for this file. Edit `integration_config.json` as appropriate for your environment.


```bash
cd ~/workspace/cf-routing-release/src/github.com/cloudfoundry-incubator/cf-routing-acceptance-tests/
cat > integration_config.json <<EOF
{
  "addresses": ["10.244.14.2"],
  "api": "api.bosh-lite.com",
  "admin_user": "admin",
  "admin_password": "admin",
  "skip_ssl_validation": true,
  "use_http":true,
  "apps_domain": "bosh-lite.com",
  "bbs_api_url": "https://bbs.service.cf.internal:8889",
  "bbs_require_ssl": true,
  "bbs_client_cert": "/path/to/bbs/client.crt",
  "bbs_client_key": "/path/to/bbs/client.key",
  "bbs_ca_cert": "/path/to/bbs/ca_cert.crt",
  "oauth": {
    "token_endpoint": "https://uaa.bosh-lite.com",
    "client_name": "tcp_emitter",
    "client_secret": "tcp-emitter-secret",
    "port": 443,
    "skip_oauth_tls_verification": true,
  }
}
EOF
export CONFIG=$PWD/integration_config.json
```

Note:
- IP `10.24.14.2` is IP address of `tcp_router_z1/0` job in cf-routing-release. If this IP address happens to be different in your cf release then change the entry accordingly.
- `admin_user` and `admin_password` properties refer to the admin user used to perform a CF login with the cf CLI.
- `skip_ssl_validation` is used for the cf CLI when targeting an environment.
- All `bbs_*` properties are only required if running the `router` test package.
- The `addresses` property contains the IP addresses of the TCP Routers and/or the Load Balancer's IP address.

BBS client cert, key and ca cert for bosh lite environment can be found in `~/workspace/cf-routing-release/src/github.com/cloudfoundry-incubator/cf-routing-acceptance-tests/assets/diego-client/config`. Replace `integration_config.json` bbs certificate fields with absolute path of certificate files.

For bosh-lite and running the router package tests, make following entry in `/etc/hosts` file
```
10.244.16.2 bbs.service.cf.internal
```
Note that IP `10.244.16.2` is IP address of `database_z1/0` job in diego release. If this IP address happens to be different in your diego release then change the entry accordingly.

### Running the tests

After correctly setting the `CONFIG` environment variable, the following command will run the tests:

```
    ./bin/test
```
