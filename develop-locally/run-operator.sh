#!/bin/sh
gcloud  container clusters get-credentials k-pipe-runner --region europe-west3
make run
exit