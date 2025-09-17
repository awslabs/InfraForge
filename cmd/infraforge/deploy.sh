#!/bin/bash
# Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

#cdk deploy --app ./infraforge
cdk deploy --app ./infraforge --force --require-approval=never
