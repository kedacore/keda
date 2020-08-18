# Release Process

The release process of a new version of KEDA involves the following: 

## 0. Prerequisites

Look at the last released version in the releases page: https://github.com/kedacore/keda/releases
For example: currently it is 1.1.0
The next version will thus be 1.2.0

## 1. Changelog

Provide a new section in `CHANGELOG.md` for the new version that is being released along with the new features, patches and deprecations it introduces. 

It should not include every single change but solely what matters to our customers, for example issue template that has changed is not important.

## 2. Create KEDA release on GitHub

Creating a new release in the releases page (https://github.com/kedacore/keda/release) will trigger a GitHub workflow which will create a new image with the latest code and tagged with the next version (in this example 1.2.0). 

KEDA Deployment YAML file (eg. keda-1.2.0.yaml) is also automatically created and attached to the Release as part of the workflow.

> Note: The Docker Hub repo with all the different images can be seen here: https://hub.docker.com/r/kedacore/keda/tags

## 3. Publish documentation for new version

Publish documentation for new version on https://keda.sh.

See [docs](https://github.com/kedacore/keda-docs#publishing-a-new-version).

## 4. Update Helm Charts

a). Update the version and appVersion here:  https://github.com/kedacore/charts/blob/master/keda/Chart.yaml 
b). In the image section update the keda and metricsAdapter to point to the docker images from step 1 https://github.com/kedacore/charts/blob/master/keda/values.yaml

Then run the commands here: https://github.com/kedacore/charts

-- To deploy KEDA through Azure Functions Core Tools --

Update the following file: 
https://github.com/Azure/azure-functions-core-tools/blob/dev/src/Azure.Functions.Cli/StaticResources/keda.yaml
[Search for 1.1.0 etc. and replace it]

## 5. Create Helm release on GitHub

Create Helm release on GitHub with changelog of what changed to our Helm chart.
