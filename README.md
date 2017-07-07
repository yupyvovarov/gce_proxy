# Proxy API to Google Compute Engine

## Description
This is a service that can be used to spawn cloud instances and configures them with a given username/password and responds with the newly created instance’s IP address.
The service listens TCP port and accept on HTTP requests.

Available next endpoints:

* GET /healthcheck

Implements a basic health check, returning HTTP status code 200 and a blank page.

* POST /v1/instances/create

Receives an username and a password as query parameters and responds with an IP address.
This endpoint creates a cloud instance, configures a user with the given username and password, enable password authentication, and add the user to sudoers.

* POST /v1/instances/ip

Receives an instance name as query parameters and responds with an IP address.

## Setup
Before run service, it should be configured in a proper way.
1. Enable the Compute Engine API in [console](https://console.developers.google.com/apis/api/compute).
2. Download your **Service account key**. [How the Application Default Credentials work](https://developers.google.com/identity/protocols/application-default-credentials#howtheywork):
  - Go to the [API Console Credentials page](https://console.developers.google.com/project/_/apis/credentials).
  - From the project drop-down, select your project.
  - On the Credentials page, select the **Create credentials** drop-down, then select **Service account  key**.
  - From the **Service account** drop-down, select an existing service account or create a new one.
  - For **Key type**, select the **JSON** key option, then select **Create**. The file automatically downloads to your computer.
  - Put the `*.json` file you just downloaded in a directory of your choosing. This directory must be private (you can't let anyone get access to this), but accessible to your web server code.
3. Create `config.json` and store in a directory of you choosing:
```json
{
  "projectid": "COMPUTE_PROJECT_NAME",
  "region": "COMPUTE_REGION",
  "zone": "COMPUTE_ZONE",
  "machinetype": "COMPUTE_MASHINE_TYPE",
  "disktype": "COMPUTE_DICK_TYPE",
  "disksize": DISK_SIZE,
  "imageid":"COMPUTE_IMAGE_PATH",
  "accountkey": "PATH_TO_SERVICE_ACCOUNT_KEY.json"
}
```
For example:
```json
{
  "projectid": "coolproject",
  "region": "us-west1",
  "zone": "us-west1-b",
  "machinetype": "f1-micro",
  "disktype": "pd-ssd",
  "disksize": 10,
  "imageid":"centos-cloud/global/images/centos-7-v20170620",
  "accountkey": "/home/tom/topsecret/coolproject-f4392646246f.json"
}
```
4. Run service. If you run service without parameters it will take `config.json` in current directory and listen port `8080`. You can change confit path and port at startup:
```bash
$ go build
$ ./gce_proxy -config /home/tom/config.json -port 8081
```

## Usage
The service accept next HTTP reqests.
1. Healthcheck request:
```bash
$ curl -X GET -I http://localhost:8080/healthcheck
```
2. Create new instance:
```bash
$ curl -X POST -d'{"username":"USERNAME","password":"PASSWORD"}' http://localhost:8080/v1/instances/create
```
3. Get IP address running instance:
```bash
$ curl -X POST -d'{"instancename":"INSTANCENAME"}' http://localhost:8080/v1/instances/ip
```
