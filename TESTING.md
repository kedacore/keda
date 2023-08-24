# Testing strategy

## Unit tests / code coverage

There are unit tests present for each scaler implementation, and for the majority of the core.
Code coverage “is something that we need to work on” constantly.

However, using a code coverage tooling is not useful for the KEDA project in the current state given a lot of functionality is covered by end-to-end tests, which are not considered and thus our code coverage metrics will be misleading.

For each PR, we automatically build and run our unit test suite but also build Docker images for both amd64 and arm64 architectures. As part of our CI process, we also perform various security checks for which you can learn more in our security section.

Lastly, we automatically perform code quality analysis with golangci-lint and check licenses of our dependencies with FOSSA.

## End-to-end tests

There are end-to-end tests for the core functionality and majority of features of KEDA as well as the scalers that it offers. These tests are required for every PR and run in the CI (however, maintainers trigger them as a security precaution). Additionally, we run our e2e test suite for every merged commit to the main branch as well as during our nightly CI schedule ([link](https://github.com/kedacore/keda/actions/workflows/nightly-e2e.yml)). Implementing end-to-end tests is a requirement for adding a new scaler, as per [our policy](https://github.com/kedacore/governance/blob/main/SCALERS.md#requirements-for-a-built-in-scaler).

The project runs two Kubernetes clusters on which all e2e tests are ran automatically.  Microsoft Azure has donated a dedicated Azure subscription so that all maintainers can manage these cloud resources and allow us to run our automated tesing and automation. This is in addition to CNCF’s Cloud Credits program which we use to provision test resources in AWS & GCP for scaler e2e tests as well.

Both the cluster management as well as the cloud resources required for our automated tests are managed by Terraform and [available on GitHub](https://github.com/kedacore/testing-infrastructure) so that every contributor can open a PR with the infrastructure changes that they require. Everything is automatically deployed by using GitHub Actions to ensure we are running the latest configuration and can easily migrate to other infrastructure, if we have to.

Additionally, CNCF sponsors KEDA by providing arm64 machines on which we build our ARM64 image and Vexxhost sponsors an OpenStack instance to run tests on these as well.
