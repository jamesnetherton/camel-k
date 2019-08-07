#!/bin/bash

# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Exit on error
set -e

source ./script/common.sh

KUBECONFIG=$(crc_kube_config)

eval $(crc oc-env)

project=$(oc project -q 2> /dev/null)
if [[ "$?" != "0" ]]; then
  echo "No project has been set. Run 'oc new-project <name>' or 'oc project <name>'"
  exit 1
fi

make images-dev-crc

./kamel install -n $project 2>/dev/null || export ret=$?

oc delete pod -l name=camel-k-operator -n $project || true
