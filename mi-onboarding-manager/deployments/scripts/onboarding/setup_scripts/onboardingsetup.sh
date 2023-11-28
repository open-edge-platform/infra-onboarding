
#!/bin/bash

#########################################################################################
# INTEL CONFIDENTIAL
# Copyright (2023) Intel Corporation
#
# The source code contained or described herein and all documents related to the source
# code("Material") are owned by Intel Corporation or its suppliers or licensors. Title
# to the Material remains with Intel Corporation or its suppliers and licensors. The
# Material contains trade secrets and proprietary and confidential information of Intel
# or its suppliers and licensors. The Material is protected by worldwide copyright and
# trade secret laws and treaty provisions. No part of the Material may be used, copied,
# reproduced, modified, published, uploaded, posted, transmitted, distributed, or
# disclosed in any way without Intel's prior express written permission.
#
# No license under any patent, copyright, trade secret or other intellectual property
# right is granted to or conferred upon you by disclosure or delivery of the Materials,
# either expressly, by implication, inducement, estoppel or otherwise. Any license under
# such intellectual property rights must be express and approved by Intel in writing.
#########################################################################################
mgr_host=$1
onb_port=$2
# Install GoLang
if [ ! -d /usr/local/go/ ]; then

   #create go-tool directory
   mkdir -p $HOME/go_tools
   pushd $HOME/go_tools > /dev/null
   wget https://dl.google.com/go/go1.20.linux-amd64.tar.gz
   tar -xzf go1.20.linux-amd64.tar.gz
   sudo mv go /usr/local/
   rm go1.20.linux-amd64.tar.gz
   sudo apt install protobuf-compiler -y >/dev/null 2>&1
   sudo apt install sshpass -y >/dev/null 2>&1
   popd > /dev/null
fi
export GOPATH=$HOME/go
export PATH=$PATH:/usr/local/go/bin:$GOPATH/bin
#copy the proto-gen-go for the compile
if [ ! -f /usr/local/go/bin/protoc-gen-go ]; then
   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

   if [ $? -eq 0 ]; then
      sudo cp $HOME/go/bin/protoc-gen-go-grpc /usr/local/go/bin/protoc-gen-go
      if [ $? -eq 1 ]; then
         echo "protoc-gen-go missing, please check"
         exit 0
      fi
   fi
fi

# Set environment variables
export PATH=$PATH:/usr/local/go/bin
export MGR_HOST=$mgr_host
export ONBMGR_PORT=$onb_port

# Run the Go program
cd ../../../../../cmd/onboardingmgr/
go run main.go
