#!/bin/sh

ls -la

# Get the value of the environment variable
PAT="$ASSETS_PAT_TOKEN"

# Clone the repository with the substituted value
git clone "https://$PAT@github.com/genshinsim/assets.git"

./preview