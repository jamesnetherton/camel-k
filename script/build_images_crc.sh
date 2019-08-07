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

tag=$1
cache_directory=$(crc_cache_directory)
ip=$(crc_ip)

# Copy build context to CRC VM
rsync -e "ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -l core -i ${cache_directory}/id_rsa_crc" -r build ${ip}:/tmp/

# Run image build
ssh -o "StrictHostKeyChecking no" -i ${cache_directory}/id_rsa_crc core@${ip} "sudo podman build -t ${tag} /tmp/build"
