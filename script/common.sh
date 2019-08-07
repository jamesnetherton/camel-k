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

crc_ip() {
  crc ip
}

crc_config_property() {
  local property=$1
  crc config view | grep ${property} | sed 's/.*\s:\s//g' | sed 's/\.[^.]*$//g'
}

crc_cache_directory() {
  local bundle=$(crc_config_property "bundle")
  echo "$(crc status | grep "Cache Directory" | awk '{print $3}')/${bundle}"
}

crc_kube_config() {
  echo "$(crc_cache_directory)/kubeconfig"
}
