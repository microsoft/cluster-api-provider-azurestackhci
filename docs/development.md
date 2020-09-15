## Testing changes

## Building and pushing dev images

1. To build images with custom tags,
   run the `make docker-build` as follows:

   ```bash
   export REGISTRY="<container-registry>"
   export MANAGER_IMAGE_TAG="<image-tag>" # optional - defaults to `dev`.
   make docker-build
   ```

2. Push your docker images:

   2.1. Login to your container registry using `docker login`.

   e.g., `docker login quay.io`

   2.2. Push to your custom image registry:

   ```bash
   REGISTRY="<container-registry>" MANAGER_IMAGE_TAG="<image-tag>" make docker-push
   ```
### Deploying with a private image.

Set your docker credentials as envvars

```bash
export DOCKER_USERNAME=<CR_USERNAME>
export DOCKER_PASSWORD=<CR_PASSWORD>
```

Set the following envvars

```bash
export AZURESTACKHCI_CLOUDAGENT_FQDN=<>
export AZURESTACKHCI_BINARY_LOCATION=<>
```

When you are ready to test your change, just run:

```bash
make deployment
```

That will take care of bringing everything up for you.

Once you are done testing with the cluster, it can be fully removed by running...

```bash
make kind-reset
```