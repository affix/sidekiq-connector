## sidekiq-connector

The sidekiq connector connects OpenFaaS functions to sidekiq topics.

Goals:

* Allow functions to subscribe to sidekiq topics
* Ingest data from sidekiq and execute functions
* Work with the OpenFaaS REST API / Gateway
* Formulate and validate a generic "connector-pattern" to be used for various event sources like sidekiq, AWS SNS, RabbitMQ etc

## Try it out

### Deploy Swarm

Deploy the stack which contains sidekiq and the connector:

```bash
docker stack deploy sidekiq -c ./yaml/connector-swarm.yml
```

* Deploy or update a function so it has an annotation `topic=faas-request` or some other topic

As an example:

```shell
$ faas store deploy figlet --annotation topic="faas-request"
```

The function can advertise more than one topic by using a comma-separated list i.e. `topic=topic1,topic2,topic3`

* Publish some messages to the topic in question i.e. `faas-request`

Instructions are below for publishing messages

* Watch the logs of the sidekiq-connector


### Deploy on Kubernetes

The following instructions show how to run `sidekiq-connector` on Kubernetes.

Deploy a function with a `topic` annotation:

```bash
$ faas store deploy figlet --annotation topic="faas-request" --gateway <faas-netes-gateway-url>
```

Deploy sidekiq:

You can run the reis, sidekiq and sidekiq-connector pods with:

```bash
kubectl apply -f ./yaml/kubernetes/
```

If you already have sidekiq then update `./yaml/kubernetes/connector-dep.yml` with your redis address and then deploy only that file:

```bash
kubectl apply -f ./yaml/kubernetes/connector-dep.yml
```

## Configuration

This configuration can be set in the YAML files for Kubernetes or Swarm.

| env_var               | description                                                 |
| --------------------- |----------------------------------------------------------   |
| `upstream_timeout`      | Go duration - maximum timeout for upstream function call    |
| `rebuild_interval`      | Go duration - interval for rebuilding function to topic map |
| `queues`                | Queues to which the connector will bind                     |
| `gateway_url`           | The URL for the API gateway i.e. http://gateway:8080 or http://gateway.openfaas:8080 for Kubernetes       |
| `redis_host`            | Default is `127.0.0.1:6379`                                          |
| `print_response`        | Default is `true` - this will output the response of calling a function in the logs |
