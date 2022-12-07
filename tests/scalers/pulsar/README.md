# Apache Pulsar Integration Tests TLS Configuration

In order to ensure the Apache Pulsar scaler correctly works with self-signed certificates, both tests are run using self-signed certs.

The Subject Alternative Name on the certs is the service name that points to the broker. Since keda runs in another namespace, it is qualified by namespace.

## Core assumptions

Here are the assumptions under which the certificates will work:

First, we need to establish the DNS names. Those are defined by the service, and will be `testName.testName`. Here are the test names:
* pulsar-partitioned-topic-test
* pulsar-non-partitioned-topic-test

Second, we must only run a single broker so that `serviceName` points only to a single broker and there are not any redirects. Given that the tests are using the standalone pulsar, it already has to be a single instance, so this assumption holds.

## Creating the self-signed certs

Generate the relevant artifacts using the following steps.

1. Generate a self-signed keystore. It has a long expiration to simplify test management.
    ```shell
    keytool \
      -keystore server.jks  -storepass protected  -deststoretype pkcs12 \
      -genkeypair -keyalg RSA -validity 36500 \
      -dname "CN=pulsar.apache.org,O=pulsar,OU=pulsar" \
      -ext "SAN=DNS:pulsar-partitioned-topic-test.pulsar-partitioned-topic-test,DNS:pulsar-non-partitioned-topic-test.pulsar-non-partitioned-topic-test"
    ```
2. Extract the public key. This will be used by the client and the server. (Requires entering the password: `protected`.)
   ```shell
   openssl pkcs12 -in server.jks -nokeys -out servercert.pem
   ```
3. Extract the private key for use by the server. (Requires entering the password: `protected`.)
   ```shell
   openssl pkcs12 -in server.jks -nodes -nocerts -out serverkey.pem
   ```
4. base64 encode `servercert.jks` and `serverkey.pem` and place them in the secret to be used in the tests. On MacOS, run:
    ```shell
    cat servercert.pem | base64 | pbcopy
    ```
    ```shell
    cat serverkey.pem | base64 | pbcopy
    ```
