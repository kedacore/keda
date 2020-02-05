# Release Process

The release process of a new version of KEDA involves the following: 

**0) Prerequisites**

Look at the last released version in the releases page: https://github.com/kedacore/keda/releases
For example: currently it is 1.1.0
The next version will thus be 1.2.0 

**1) Update KEDA version in code and YAMLs**

Update version string in:
https://github.com/kedacore/keda/blob/master/version/version.go 

Update sections related to images in these yamls:
https://github.com/kedacore/keda/blob/master/deploy/12-operator.yaml
https://github.com/kedacore/keda/blob/master/deploy/22-metrics-deployment.yaml

Update sections related to the version label in these yamls:
https://github.com/kedacore/keda/blob/master/deploy/00-namespace.yaml
https://github.com/kedacore/keda/blob/master/deploy/01-service_account.yaml
https://github.com/kedacore/keda/blob/master/deploy/10-cluster_role.yaml
https://github.com/kedacore/keda/blob/master/deploy/11-role_binding.yaml
https://github.com/kedacore/keda/blob/master/deploy/12-operator.yaml
https://github.com/kedacore/keda/blob/master/deploy/20-metrics-cluster_role.yaml
https://github.com/kedacore/keda/blob/master/deploy/21-metrics-role_binding.yaml
https://github.com/kedacore/keda/blob/master/deploy/22-metrics-deployment.yaml
https://github.com/kedacore/keda/blob/master/deploy/23-metrics-service.yaml
https://github.com/kedacore/keda/blob/master/deploy/24-metrics-api_service.yaml

Commit these changes.

**2) Deploy the new KEDA images to Docker Hub**

The Docker Hub repo with all the different images can be seen here: https://hub.docker.com/r/kedacore/keda/tags

Creating a new release in the the releases page (https://github.com/kedacore/keda/release) will trigger a GitHub workflow which will create a new image with the latest code and tagged with the next version (in this example 1.2.0) and also change the latest tag to point to this image as well. 

**3) Update Helm Charts**

a). Update the version and appVersion here:  https://github.com/kedacore/charts/blob/master/keda/Chart.yaml 
b). In the image section update the keda and metricsAdapter to point to the docker images from step 1 https://github.com/kedacore/charts/blob/master/keda/values.yaml

Then run the commands here: https://github.com/kedacore/charts

-- To deploy KEDA through Azure Functions Core Tools --

Update the following file: 
https://github.com/Azure/azure-functions-core-tools/blob/dev/src/Azure.Functions.Cli/StaticResources/keda.yaml
[Search for 1.1.0 etc. and replace it]
