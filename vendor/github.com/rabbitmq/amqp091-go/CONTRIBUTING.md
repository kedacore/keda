# Contributing

## Workflow

Here is the recommended workflow:

1. Fork this repository, **github.com/rabbitmq/amqp091-go**
1. Create your feature branch (`git checkout -b my-new-feature`)
1. Run Static Checks
1. Run integration tests (see below)
1. **Implement tests**
1. Implement fixs
1. Commit your changes (`git commit -am 'Add some feature'`)
1. Push to a branch (`git push -u origin my-new-feature`)
1. Submit a pull request

## Running Static Checks

golangci-lint must be installed to run the static checks. See [installation
docs](https://golangci-lint.run/usage/install/) for more information.

The static checks can be run via:

```shell
make checks
```

## Running Tests

### Integration Tests

Running the Integration tests require:

* A running RabbitMQ node with all defaults:
  [https://www.rabbitmq.com/download.html](https://www.rabbitmq.com/download.html)
* That the server is either reachable via `amqp://guest:guest@127.0.0.1:5672/`
  or the environment variable `AMQP_URL` set to it's URL
  (e.g.: `export AMQP_URL="amqp://guest:verysecretpasswd@rabbitmq-host:5772/`)

The integration tests can be run via:

```shell
make tests
```

All integration tests should use the `integrationConnection(...)` test
helpers defined in `integration_test.go` to setup the integration environment
and logging.
