#!/bin/bash
#####################################################
#
# This script to the the Image URL from DKAM Server
#
#####################################################

#set -x
DKAM_BASE_DIR=""
#Check if go lang tools installed on the machine,if not install it
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

#clone the DKAM proto code and compile for calling the Base image URL
if [ ! -d ../../../../../proto/ ]; then
   echo "procto directory not found for building the dkam"
   exit 0
else
   DKAM_BASE_DIR=../../../../..
   pushd $DKAM_BASE_DIR/proto/ > /dev/null

   protoc --proto_path=. --go_out=. --go-grpc_out=. dkam.proto
   popd >/dev/null
   pushd $DKAM_BASE_DIR/helm/edge-iaas-platform/platform-director/dkammgr/mongodb > /dev/null
   #pull mongodb if its not present
   if [[ ! $(docker ps -f "name=mongodb" --format '{{.Names}}') == mongodb ]]; then

      docker run -v $PWD/data:/data/db -v $PWD/mongorestore.sh:/docker-entrypoint-initdb.d/mongorestore.sh -d -p 27017:27017 --name mongodb mongo:latest
   fi
   popd >/dev/null
   pushd $DKAM_BASE_DIR/cmd/dkammgr >/dev/null

   pid=$(go run ./main.go &) &  >/dev/null >/dev/null 2>&1
   if [ $? -eq 1 ]; then
      echo "Unable to connect to MangoDB please check!!!!"
      exit 0
   fi
   echo "Success"
   sleep 25
   popd >/dev/null
fi
#call the DKAM API to get the base os URL
pushd $DKAM_BASE_DIR/cmd/dkamagent > /dev/null

Base_OS_URL=$(go run main.go 2>&1)
Base_OS_URL=$(echo $Base_OS_URL | awk '{print $4}')

chars=$(echo -n "$Base_OS_URL" | wc -m)
Base_OS_URL=$(echo $Base_OS_URL | cut -c 15-$((chars - 1)))

popd > /dev/null
export bkc_url=$Base_OS_URL
