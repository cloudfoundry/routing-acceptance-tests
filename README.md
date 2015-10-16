# Router Acceptance Tests 

This test suite exercises [Router](https://github.com/cloudfoundry-incubator/cf-routing-release).

## Running Acceptance tests

### Test setup

To run the Router Acceptance tests, you will need:
- a running router deployment
- make sure that all the dependencies of this module are installed in your GOPATH (alternatively if you have cloned the [cf-routing-release](https://github.com/cloudfoundry-incubator/cf-routing-release) make sure that your GOPATH is set to root directory of cf-routing-release)
- an environment variable `ROUTER_API_CONFIG` which points to a `.json` file that contains the router api endpoint

The following commands will create a config file `router_config.json` for a [bosh-lite](https://github.com/cloudfoundry/bosh-lite) installation and set the `ROUTER_API_CONFIG` environment variable to the path for this file. Edit `router_config.json` as appropriate for your environment.


```bash
cd ~/workspace/cf-routing-release/src/github.com/cloudfoundry-incubator/cf-tcp-router-acceptance-tests/
cat > router_config.json <<EOF
{
  "addresses": ["10.244.8.2"],
  "port": 9999,
  "bbs_api_url": "https://bbs.service.cf.internal:8889",
  "bbs_require_ssl": true,
  "bbs_client_cert": "/path/to/bbs/client.crt",
  "bbs_client_key": "/path/to/bbs/client.key",
  "bbs_ca_cert": "/path/to/bbs/ca_cert.crt",
  "routing_api_url": "http://10.244.0.134:3000",
  "oauth": {
    "token_endpoint": "http://uaa.bosh-lite.com",
    "client_name": "gorouter",
    "client_secret": "gorouter-secret",
    "port": 80
  }
}
EOF
export ROUTER_API_CONFIG=$PWD/router_config.json
```
BBS client cert, key and ca cert for bosh lite environment can be found in `~/workspace/cf-routing-release/src/github.com/cloudfoundry-incubator/cf-tcp-router-acceptance-tests/assets/desired_lrp_client/config`. Replace `router_config.json` bbs certificate fields with absolute path of certificate files.

Note that IP `10.24.0.134` is IP address of `api_z1/0` job in cf release. If this IP address happens to be different in your cf release then change the entry accordingly.

Make following entry in `/etc/hosts` file
```
10.244.16.130 bbs.service.cf.internal
```
Note that IP `10.244.16.130` is IP address of `database_z1/0` job in diego release. If this IP address happens to be different in your diego release then change the entry accordingly.

### Running the tests

After correctly setting the `ROUTER_API_CONFIG` environment variable, the following command will run the tests:

```
    ginkgo -r -nodes=3 router
```
