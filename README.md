The python `gcloud` has always sucked. It's slow, takes up a ~100MB of disk space, and annoying to install in CI/CD environments. But I've always tolerated it because it worked just well enough. That has changed recently with some super annoying changes ([1](https://issuetracker.google.com/issues/224754679), [2](https://cloud.google.com/blog/products/containers-kubernetes/kubectl-auth-changes-in-gke)) to `gcloud`. So let's fix it by reimplementing `gcloud` in go!

This implementation of `gcloud` is incrementally adoptable, meaning that it will automatically fallback to python `gcloud` if we have not implemented a specific subcommand. My initial goal is to implement all the things I personally need in CI/CD environments for linux environments. Contributions are accepted for other features and environments.

## Install

Regardless of your installation method, you need to make sure that this `gcloud` is higher in your `$PATH` than the python `gcloud`.

A good `$PATH` looks like this (where this `gcloud` is in `/home/alex/go/bin`):

```
echo $PATH
/home/alex/go/bin:/opt/google-cloud-sdk/bin/
```

### Install with go

```
go install github.com/gartnera/gcloud@latest
```

### Install from release

```
curl -L https://github.com/gartnera/gcloud/releases/download/v0.0.1/gcloud_0.0.1_linux_amd64.tar.gz | tar xz -C /usr/local/bin
```

or

```
wget -O - https://github.com/gartnera/gcloud/releases/download/v0.0.1/gcloud_0.0.1_linux_amd64.tar.gz | tar xz -C /usr/local/bin
```
