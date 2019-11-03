+++
title = "FAQ"
date = "2017-10-05"
fragment = "content"
weight = 100
+++

#### What is KEDA and why is it useful?
KEDA stands for Kubernetes Event Driven Auto-Scaler. It is built to be able to activate a Kubernetes deployment (i.e. no pods to a single pod) and subsequently to more pods based on events from various event sources.

#### What are the prerequisites for using KEDA?
KEDA is designed to be run on any vanilla Kubernetes cluster. It uses a CRD and needs a Metric Server so you will have to use a Kubernetes version which supports these. Any Kubernetes cluster >= 1.11.10 have been tested and should work.

#### Does KEDA depend on any Azure service?
No, KEDA only takes a dependency on vanilla Kubernetes constructs and can run on any Kubernetes cluster whether in OpenShift, AKS, GKE, EKS or your own.

#### Does KEDA only work with Azure Functions?
No, KEDA can scale up/down any container that you specify in your deployment. There has however been work done in the Azure Function tooling to make it easy to install an Azure Function container.

#### Why should we use KEDA if we are already use Azure Functions in Azure?

* Want to run functions on-premises (potentially in something like an 'intelligent edge' architecture)
* Want to run functions alongside other Kubernetes apps (maybe in a restricted network, app mesh, custom environment, etc.)
* Want to run functions outside of Azure (no vendor lockin)
* Specific need for more control (GPU enabled compute clusters, more knobs and such)

#### Can I scale my Http triggered Azure Function on Kubernetes with KEDA?
Yes we currently enable this through Osiris. It is integrated in the Azure Functions tooling i.e. through the Core tools. We are working on making significant investment in this scenario going forward. Stay tuned!

How do I add a new Scaler?
[TODO]

####  Where can I get to the code for the Scalers?
All scalers have their code [here](https://github.com/kedacore/keda/tree/master/pkg/scalers)

#### Is short polling intervals a problem?
Polling interval really only impacts the time-to-activation (scaling from 0 to 1) but once scaled to one it's really up to the HPA which polls KEDA.

#### How can I get involved?
There are several ways to get involved.

* Pick up an issue to work on. A good place to start might be issues which are marked as [Good First Issue](https://github.com/kedacore/keda/labels/good%20first%20issue) or [Help Wanted](https://github.com/kedacore/keda/labels/help%20wanted)
* We are always looking to add more scalers. These are some ideas of missing scalers
* We are always looking for more samples, documentation etc.
* Please join us in our [weekly standup](https://github.com/kedacore/keda#community-standup)

#### Can KEDA be used in production?
As of September 2019 KEDA is still in beta and hence it is not suggested to use in production. We are fast approaching a 1.0 release and then you should be able to use it in production.

#### Is there an ETA for KEDA release 1.0?
Currently we are targeting the end of 2019 if not earlier for a KEDA 1.0 release.

#### What does it cost?
There is no charge for using KEDA itself.