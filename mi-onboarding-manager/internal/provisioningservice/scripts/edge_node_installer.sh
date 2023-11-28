#!/bin/bash
#######################################################
#
# This is to pull the edge node docker images to OS and
# start the agents
#
#######################################################

AGENT_LOG_FILENAME="edge_node_agent_log.txt"
SCRIPT_DIR=$(pwd)

touch $SCRIPT_DIR/$AGENT_LOG_FILENAME

source /home/user/.bashrc

# check docker compose v1 or v2
if ! command -v docker-compose >/dev/null 2>&1; then
    dc='docker compose'
else
    dc='docker-compose'
fi

#Install the Intel certificates before pulling the docker images
function install_intel_cacerts()
{
	#download the cacerts
	wget https://ubit-artifactory-or.intel.com/artifactory/it-btrm-local/intel_cacerts/install_intel_cacerts_linux.sh

	if  [ -f install_intel_cacerts_linux.sh ]; then 
	    wget https://ubit-artifactory-or.intel.com/artifactory/it-btrm-local/intel_cacerts/install_intel_cacerts_linux.sh --no-proxy
	fi

	chmod +x install_intel_cacerts_linux.sh

	sudo ./install_intel_cacerts_linux.sh

	if [ $? -eq 0 ]; then 
		echo "Intel cacertificates installed successfully" >> $SCRIPT_DIR/$AGENT_LOG_FILENAME
		sudo systemctl restart docker
		sudo systemctl daemon-reload
	else
		echo "Intel cacertificates installed failed" >> $SCRIPT_DIR/$AGENT_LOG_FILENAME
		exit 0
	fi
}

#update docker proxy for PD IP and Node IP
function update_the_docker_proxy()
{
    PD_IP=$(cat /home/user/.bashrc | grep UPDATEMGR_HOST | cut -d"=" -f2)
    NODE_IP=$(hostname -I | tr ' ' '\n' | grep -v "^127" | head -n 1)
    sed -i "s/gar-registry.caas.intel.com/& ,$PD_IP,$NODE_IP/" /etc/systemd/system/docker.service.d/http-proxy.conf
    sudo systemctl daemon-reload
    sudo systemctl restart docker
}

#Install Inventory Agent 
function install_inventory_agent()
{
    #create a directry for inventory agent 
    
    cd $SCRIPT_DIR/inv_agent
    #pull the inventory agent image
    $dc --env-file $SCRIPT_DIR/agent_node_env.txt up -d 

    if [ $? -eq 0 ]; then 
	echo "successfully started the update  agent container" >> ../$AGENT_LOG_FILENAME
    else
        echo "failed to start the update agent container please check" >> ../$AGENT_LOG_FILENAME
    fi
    cd -
}

#Install Update Manager Agent
function install_updatemgr_agent()
{
    #create a directry for update manager agent
    cd $SCRIPT_DIR/upd_mgr_agent

    #pull the update manager agent image
    $dc --env-file $SCRIPT_DIR/agent_node_env.txt up -d

    if [ $? -eq 0 ]; then
        echo "age" >> ../$AGENT_LOG_FILENAME
    else
        echo "failed to pulle the update manager agent image please check" >> ../$AGENT_LOG_FILENAME

    fi
    cd -
}

#Install telemetry  Agent
function install_telemetry_agent()
{
    #create a directry for telemetry agent
    cd $SCRIPT_DIR/telmtry_agent

    #untar the telemetry deb files
    tar -xvf telemetry_agent_deb_pkg.tar

    cd telemetry_agent_deb_pkg

    #install the debian pkgs 

    dpkg -i iaas-telemetry_0.3.8_amd64.deb iaas-telemetry-otelcol_0.3.8_amd64.deb iaas-telemetry-telemetry-agent_0.3.8_amd64.deb
    if [ $? -eq 0 ]; then
        echo "Successfully installed the telemetry agent deb files" >> ../../$AGENT_LOG_FILENAME
    else
	echo "Failed to installed the telemetry agent deb files,please check" >> ../../$AGENT_LOG_FILENAME
    fi

    #update the config.yaml file

    sleep 5 

    PD_IP=$(cat /home/user/.bashrc | grep UPDATEMGR_HOST | cut -d"=" -f2)

    eval "sed -i 's/localhost/${PD_IP}/g' /run/iaas-telemetry/config.yaml"
    sed -i "s/5566/30205/g" /run/iaas-telemetry/config.yaml

    eval "sed -i 's/localhost/${PD_IP}/g' /run/iaas-telemetry/iaas-telemetry/telemetry-agent/config.yaml"
    eval "sed -i 's/telemetry.intel.pd:10.49.76.146/telemetry.intel.pd:${PD_IP}/g' /run/iaas-telemetry/iaas-telemetry/telemetry-agent/docker-compose.yml"
    eval "sed -i 's//telemetry.intel.pd:10.49.76.146/telemetry.intel.pd:${PD_IP}/g' /run/iaas-telemetry/iaas-telemetry/otelcol/docker-compose.yml"
    sed -i "s/5566/30205/g" /run/iaas-telemetry/iaas-telemetry/telemetry-agent/config.yaml 
    eval "sed -i 's/localhost/${PD_IP}/g' /usr/share/iaas-telemetry/telemetry-agent/config.yaml"
    sed -i "s/5566/30205/g" /usr/share/iaas-telemetry/telemetry-agent/config.yaml
    sudo systemctl restart iaas-telemetry-telemetry-agent.service

    cd $SCRIPT_DIR
    

}
###Main####

#install_intel_cacerts


install_inventory_agent

update_the_docker_proxy

install_updatemgr_agent

install_telemetry_agent

touch $SCRIPT_DIR/.agent_install_done





