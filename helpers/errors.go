package helpers

import "errors"

var ErrFallback = errors.New("falling back to python gcloud")
var ErrFallbackNoToken = errors.New("falling back to python gcloud with no token")
