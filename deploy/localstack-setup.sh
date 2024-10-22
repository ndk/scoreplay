#!/bin/bash

set -e

awslocal cloudformation create-stack --stack-name scoreplay-stack --template-body file:///scoreplay-cfn.yaml