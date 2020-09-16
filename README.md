# Kubernetes Cluster API Provider Azure Stack HCI

<img src="https://github.com/kubernetes/kubernetes/raw/master/logo/logo.png"  width="100">

------

Kubernetes-native declarative infrastructure for Azure Stack HCI.

## What is the Cluster API Provider Azure Stack HCI

The [Cluster API][cluster_api] brings declarative, Kubernetes-style APIs to cluster creation, configuration and management.

The API itself is shared across multiple cloud providers allowing for true Azure Stack HCI
hybrid deployments of Kubernetes.

## Quick Start

Check out the [Cluster API Quick Start][quickstart] to create your first Kubernetes cluster on Azure Stack HCI using Cluster API.

---

## Support Policy

This provider's versions are compatible with the following versions of Cluster API:

|  | Cluster API `v1alpha1` (`v0.1.x`) | Cluster API `v1alpha2` (`v0.2.x`) | Cluster API `v1alpha3` (`v0.3.x`) |
|---|---|---|---|
|AzureStackHCI Provider `v0.2.x` |  | ✓ |  |
|AzureStackHCI Provider `v0.3.x` |  |  | ✓ |

This provider's versions are able to install and manage the following versions of Kubernetes:

|  | AzureStackHCI Provider `v0.2.x` | AzureStackHCI Provider `v0.3.x`
|---|---|---|
| Kubernetes 1.16 | ✓ | ✓ |
| Kubernetes 1.17 | ✓ | ✓ |
| Kubernetes 1.18 | ✓ | ✓ |


Each version of Cluster API for Azure Stack HCI will attempt to support at least two Kubernetes versions e.g., Cluster API for Azure Stack HCI `v0.2` may support Kubernetes 1.14 and Kubernetes 1.15.

**NOTE:** As the versioning for this project is tied to the versioning of Cluster API, future modifications to this policy may be made to more closely align with other providers in the Cluster API ecosystem.

---

## Documentation

Documentation is in the `/docs` directory, and the [index is here](docs/README.md).

## Contributing

This project welcomes contributions and suggestions.  Most contributions require you to agree to a
Contributor License Agreement (CLA) declaring that you have the right to, and actually do, grant us
the rights to use your contribution. For details, visit https://cla.opensource.microsoft.com.

When you submit a pull request, a CLA bot will automatically determine whether you need to provide
a CLA and decorate the PR appropriately (e.g., status check, comment). Simply follow the instructions
provided by the bot. You will only need to do this once across all repos using our CLA.

This project has adopted the [Microsoft Open Source Code of Conduct](https://opensource.microsoft.com/codeofconduct/).
For more information see the [Code of Conduct FAQ](https://opensource.microsoft.com/codeofconduct/faq/) or
contact [opencode@microsoft.com](mailto:opencode@microsoft.com) with any additional questions or comments.

## Github issues

### Bugs

If you think you have found a bug please follow the instructions below.

- Please spend a small amount of time giving due diligence to the issue tracker. Your issue might be a duplicate.
- Get the logs from the cluster controllers. Please paste this into your issue.
- Open a [bug report][bug_report].
- Remember users might be searching for your issue in the future, so please give it a meaningful title to helps others.
- Feel free to reach out to the cluster-api community on [kubernetes slack][slack_info].

### Tracking new features

We also use the issue tracker to track features. If you have an idea for a feature, or think you can help ClusterAPI become even more awesome follow the steps below.

- Open a [feature request][feature_request].
- Remember users might be searching for your issue in the future, so please
  give it a meaningful title to helps others.
- Clearly define the use case, using concrete examples. EG: I type `this` and
  cluster-api-provider-azurestackhci does `that`.
- Some of our larger features will require some design. If you would like to
  include a technical design for your feature please include it in the issue.
- After the new feature is well understood, and the design agreed upon we can
  start coding the feature. We would love for you to code it. So please open
  up a **WIP** *(work in progress)* pull request, and happy coding.

<!-- References -->

[bug_report]: https://github.com/microsoft/cluster-api-provider-azurestackhci/issues/new?template=bug_report.md
[feature_request]: https://github.com/microsoft/cluster-api-provider-azurestackhci/issues/new?template=feature_request.md
[cluster_api]: https://github.com/kubernetes-sigs/cluster-api
[quickstart]: https://cluster-api.sigs.k8s.io/user/quick-start.html
[slack_info]: https://kubernetes.slack.com/archives/C8TSNPY4T