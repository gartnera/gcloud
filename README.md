The python `gcloud` has always sucked. It's slow, takes up a ~100MB of disk space, and annoying to install in CI/CD environments. But I've always tolerated it because it worked just well enough. That has changed recently with some super annoying changes ([1](https://issuetracker.google.com/issues/224754679), [2](https://cloud.google.com/blog/products/containers-kubernetes/kubectl-auth-changes-in-gke)). So let's fix it by reimplementing `gcloud` in go!

This implementation of `gcloud` is incrementally adoptable, meaning that it will automatically fallback to python `gcloud` if we have not implemented a specific subcommand. My initial goal is to implement things I personally need for CI/CD in Linux environments. Contributions are welcome for other features and environments.

## Install

Regardless of your installation method, you need to make sure that this `gcloud` is higher in your `$PATH` than the python `gcloud`.

A good `$PATH` looks like this (where this `gcloud` is in `/home/alex/go/bin`):

```
echo $PATH
/home/alex/go/bin:/opt/google-cloud-sdk/bin/
```

If you do not install via the release, you must manually install `scripts/docker-credential-gcloud`.

### Install from release

```
curl -L https://github.com/gartnera/gcloud/releases/download/v0.0.7/gcloud_0.0.7_linux_amd64.tar.gz | tar xz -C /usr/local/bin
```

or

```
wget -O - https://github.com/gartnera/gcloud/releases/download/v0.0.7/gcloud_0.0.7_linux_amd64.tar.gz | tar xz -C /usr/local/bin
```

### Install with go

```
go install github.com/gartnera/gcloud@latest
```

## Supported Commands

- `gcloud auth application-default login` (code flow)
- `gcloud auth application-default print-access-token`
- `gcloud auth configure-docker`
- `gcloud auth docker-helper`

- `gcloud container clusters get-credentials`
- `gcloud config config-helper --format=client.authentication.k8s.io/v1` (used by `gcloud container clusters get-credentials`)
