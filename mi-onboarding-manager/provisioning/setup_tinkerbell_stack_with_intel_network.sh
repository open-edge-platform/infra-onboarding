#!/bin/bash
########################################################################################################
#
#  This file is for automating the installation of Tinkerbell and related packages for IaaS  Project WS3.
#  This covers Setup Host with
#     Proxy Settings, Kubernetes, RKE2 Cluster, tinkerbell stack ,ect ..
#  Author: Jayanthi, Shankar Srinivas  <shankar.srinivas.jayanthi@intel.com>
#
########################################################################################################
#set -x

source config

SETUP_STATUS_FILENAME=".tinkerbell_setup_status"
SCRIPT_DIR=$(pwd)

touch $SCRIPT_DIR/$SETUP_STATUS_FILENAME

##############################Fixed Variables####################################
update_etc_env="""http_proxy=$http_proxy\nhttps_proxy=$https_proxy\nftp_proxy=$ftp_proxy\nsocks_proxy=$socks_proxy\nrsync_proxy=$http_proxy\nno_proxy=$no_proxy\n"""

docker_proxy_conf="""[Service]\nEnvironment=\"HTTP_PROXY=$http_proxy\"\nEnvironment=\"HTTPS_PROXY=$https_proxy\"\nEnvironment=\"NO_PROXY=$no_proxy\"\n"""

rke_proxy_conf="""[Service]\nHTTP_PROXY=$http_proxy\nHTTPS_PROXY=$https_proxy\nNO_PROXY=$no_proxy\n"""

##############################Functions#########################################

#Check for the global variables assignments
function check_pre_condition() {
	if grep -q "check_pre_conditio done" $SCRIPT_DIR/$SETUP_STATUS_FILENAME; then
		echo "Skipping check_pre_condition"
	else
		if [ -z $load_balancer_ip ] || [ -z $tinkerbell_charts_stable_branch ]; then
			echo "Please provide the interface details for public loadBalancerIP "
			exit 0
		fi
		echo "check_pre_conditio done" >>$SCRIPT_DIR/$SETUP_STATUS_FILENAME
	fi
}

#Setup the Proxy Settings
function setup_system_proxy() {
	apt_update=""
	env_update=""
	docker_proxy=""
	if grep -q "setup_system_proxy done" $SCRIPT_DIR/$SETUP_STATUS_FILENAME; then
		echo "Skipping setup_proxy"
	else
		#update apt.conf proxy settings
		if [ ! -f /etc/apt/apt.conf ] || [ ! -s /etc/apt/apt.conf ]; then
			local proxy="Acquire::http::Proxy \"$http_proxy\";"
			echo $proxy | sudo tee -a /etc/apt/apt.conf >/dev/null
			local proxy="Acquire::https::Proxy \"$https_proxy\";"
			echo $proxy | sudo tee -a /etc/apt/apt.conf >/dev/null
			apt_update=1

		fi

		#update the enviromnent proxy settings
		sudo bash -c ">/etc/environment"
		if grep -q "http_proxy*" "/etc/environment"; then
			echo "Proxy already added in /etc/environment"
		else
			echo -e $update_etc_env | sudo tee -a /etc/environment >/dev/null
			env_update=1
		fi
		if [ ! -f /etc/systemd/system/docker.service.d/proxy.conf ] || [ ! -s /etc/systemd/system/docker.service.d/proxy.conf ]; then
			#create the docker service directory
			sudo mkdir -p /etc/systemd/system/docker.service.d
			echo -e $docker_proxy_conf | sudo tee /etc/systemd/system/docker.service.d/proxy.conf >/dev/null

			docker_proxy=1
		fi

		if [ $apt_update ] || [ $env_update ] || [ $docker_proxy ]; then
			echo "Environment varaibles updated please reboot the machine and re run the script after system boots up"
			echo "setup_system_proxy done" >>$SCRIPT_DIR/$SETUP_STATUS_FILENAME
			echo "====SYSTEM PROXY SETUP DONE======="
			exit 0
		fi
		echo "====SYSTEM PROXY SETUP DONE======="
		echo "setup_system_proxy done" >>$SCRIPT_DIR/$SETUP_STATUS_FILENAME
	fi
}
#This is get the system ip details
function get_system_ip_details() {
	sudo apt install net-tools -y
	pub_inerface_name=$(route | grep '^default' | grep -o '[^ ]*$')
	pd_host_ip=$(ifconfig "${pub_inerface_name}" | grep 'inet ' | awk '{print $2}')
}
#Update the system after proxy settings
function update_system_packages() {
	sudo apt update
	if [ $? -ne 0 ]; then
		echo "something wrong the the system please check before proceeing further"
		exit 0
	fi
}
#Install docker Services
function install_docker_services() {
	if grep -q "install_docker_services  done" $SCRIPT_DIR/$SETUP_STATUS_FILENAME; then
		echo "Skipping  install_docker_services"
	else
		sudo apt update
		sudo apt install ca-certificates curl gnupg -y
		status1=$(echo $?)

		sudo install -m 0755 -d /etc/apt/keyrings
		status2=$(echo $?)

		curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg

		status3=$(echo $?)

		sudo chmod a+r /etc/apt/keyrings/docker.gpg

		if [ $status1 -ne 0 && $status2 -ne 0 && $status3 -ne 0 ]; then
			echo "Something went wrong in docker setup please check!!"
			echo "install_docker_services  not done correctly please re check" >>$SCRIPT_DIR/$SETUP_STATUS_FILENAME
			exit 0
		else

			echo \
				"deb [arch="$(dpkg --print-architecture)" signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
                "$(. /etc/os-release && echo "$VERSION_CODENAME")" stable" |
				sudo tee /etc/apt/sources.list.d/docker.list >/dev/null
			sudo apt update
			sudo apt install docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin -y

			echo "====DOCKER  SETUP DONE======="
			echo "install_docker_services  done" >>$SCRIPT_DIR/$SETUP_STATUS_FILENAME

			sudo usermod -aG docker $USER
			sudo systemctl restart docker
			sudo systemctl daemon-reload
			sudo chmod 666 /var/run/docker.sock

		fi

	fi
}
#Install kubecontroler
function install_kubectl_service() {
	if grep -q "install_kubectl_service  done" $SCRIPT_DIR/$SETUP_STATUS_FILENAME; then
		echo "Skipping  install_kubectl_service"
	else
		curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"

		if [ $? -ne 0 ]; then
			echo "Downloding kubectl Failed please check!"
			echo "Downloding kubectl Failed please check!" >>$SCRIPT_DIR/$SETUP_STATUS_FILENAME
			exit 0
		else
			chmod +x ./kubectl

			sudo mv ./kubectl /usr/local/bin/kubectl

			echo "====KUBCTL SETUP DONE======="
			echo "install_kubectl_service  done" >>$SCRIPT_DIR/$SETUP_STATUS_FILENAME
		fi
	fi
}
#Set the rke2 proxy settings
function set_rk2_proxy() {

	if grep -q "set_rk2_proxy done" $SCRIPT_DIR/$SETUP_STATUS_FILENAME; then
		echo "Skipping set_rk2_proxy"
	else
		http_proxy=$(sudo cat /etc/environment | grep http_proxy | awk -F "=" '{print $2}')
		https_proxy=$(sudo cat /etc/environment | grep https_proxy | awk -F "=" '{print $2}')
		no_proxy=$(sudo cat /etc/environment | grep no_proxy | awk -F "=" '{print $2}')

		echo -e $rke_proxy_conf | sudo tee /etc/default/rke2-server >/dev/null
   
		if [ ! -f /root/.docker/config.json ] || [ ! -s /root/.docker/config.json ]; then
			sudo mkdir -p /root/.docker/
			cat <<EOF | sudo tee /root/.docker/config.json
{
 "proxies":
 {
  "default":
  {
   "httpProxy": "$http_proxy",
   "httpsProxy": "$https_proxy",
    "noProxy": "$no_proxy"
  }
 }
}
EOF
		fi

		echo "set_rk2_proxy done" >>$SCRIPT_DIR/$SETUP_STATUS_FILENAME

	fi
}
#Install RKE2 services
function create_rk2e_services() {

	if grep -q "create_rk2e_services done" $SCRIPT_DIR/$SETUP_STATUS_FILENAME; then
		echo "Skipping create_rk2e_services"
	else
		#Please check if its already running
		sudo systemctl status rke2-server.service | grep -i "Active:" | grep running >/dev/null 2>&1
		if [ $? -eq 0 ]; then
			echo "rke2 services are already running no need to start again"
			echo "create_rk2e_services done" >>$SCRIPT_DIR/$SETUP_STATUS_FILENAME
		else
			if [ -f /usr/local/bin/rke2-killall.sh ]; then
				sudo /usr/local/bin/rke2-killall.sh >/dev/null 2>&1
			fi

			if [ -f /usr/local/bin/rke2-uninstall.sh ]; then
				sudo /usr/local/bin/rke2-uninstall.sh >/dev/null 2>&1
			fi
			sleep 2
			sudo bash -c 'curl -sfL https://get.rke2.io | INSTALL_RKE2_VERSION=v1.25.10+rke2r1  sh -'
			sudo systemctl enable rke2-server.service
			sleep 2
			sudo systemctl start rke2-server.service

			if [ $? -ne 0 ]; then
				echo "create_rk2e_services Failed please check!"
				echo "create_rk2e_services Failed please check!!" >>$SCRIPT_DIR/$SETUP_STATUS_FILENAME
				exit 0
			else
				mkdir /home/$USER/.kube
				sudo cp /etc/rancher/rke2/rke2.yaml /home/$USER/.kube/config
				sudo chmod 755 /etc/rancher/rke2/rke2.yaml
				sudo echo "chmod 755 /etc/rancher/rke2/rke2.yaml" >>~/.profile
				sudo chmod 755 /home/$USER/.kube/config
				sudo chown $USER:$USER /home/$USER/.kube/config
				echo "create_rk2e_services done" >>$SCRIPT_DIR/$SETUP_STATUS_FILENAME
			fi

		fi
	fi
}
#Installing Helm services
function install_helm_service() {
	if grep -q "install_helm_service done" $SCRIPT_DIR/$SETUP_STATUS_FILENAME; then
		echo "Skipping install_helm_service"
	else
		export http_proxy=$http_proxy
		export https_proxy=$https_proxy
		curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3
		if [ $? -ne 0 ]; then
			echo "Downloading helm package Failed please check!"
			echo "Downloading helm package Failed please check!!" >>$SCRIPT_DIR/$SETUP_STATUS_FILENAME
			exit 0

		else
			chmod 700 get_helm.sh
			./get_helm.sh --version v3.12.3
			if [ $? -ne 0 ]; then
				echo "Helm Installation Failed please check!"
				echo "Helm Installation Failed please check!!" >>$SCRIPT_DIR/$SETUP_STATUS_FILENAME
				exit 0
			else
				echo "====HELM SETUP DONE======="
				echo "install_helm_service done" >>$SCRIPT_DIR/$SETUP_STATUS_FILENAME
			fi

		fi

	fi
}

#Installing tinkerbell stack
function install_tinkerbell_stack() {
	if grep -q "install_tinkerbell_stack  done" $SCRIPT_DIR/$SETUP_STATUS_FILENAME; then
		echo "Skipping  install_tinkerbell_stack"
	else
		#clone the tinkerbell charts
		#before clone just check if already charts directory present or not , if yes rename it
		if [ -d $SCRIPT_DIR/charts ]; then
			if [ -d $SCRIPT_DIR/charts.org ]; then
				sudo rm -rf $SCRIPT_DIR/charts.org
				sudo mv $SCRIPT_DIR/charts $SCRIPT_DIR/charts.org
			else
				sudo mv $SCRIPT_DIR/charts $SCRIPT_DIR/charts.org
			fi
		fi
		git clone https://github.com/tinkerbell/charts.git
		cd $SCRIPT_DIR/charts

		#check out to a stable branch branch
		git checkout $tinkerbell_charts_stable_branch
		cd -

		if [ $? -ne 0 ]; then
			echo "git clone faild for tinkerbell charts please check!"
			echo "git clone faild for tinkerbell charts please check!" >>$SCRIPT_DIR/$SETUP_STATUS_FILENAME

		else
			#check tinkstack is running or not , if running un install helm and start it
			sudo chmod 755 /etc/rancher/rke2/rke2.yaml
			kubectl get all --all-namespaces | grep "service/tink-stack"
			if [ $? -eq 0 ]; then
				echo "tinkstack is already running , removing it before going further"
				helm uninstall stack-release --namespace tink-system
			fi

			#update hook for using intel DHCP rather boots DHCP procedure
			sed -i 's/url: https:\/\/github.com\/tinkerbell\/hook\/releases\/download\/v0.8.0\/hook_x86_64.tar.gz/url: https:\/\/github.com\/tinkerbell\/hook\/releases\/download\/v0.8.1\/hook_x86_64.tar.gz/g' charts/tinkerbell/stack/values.yaml
			sed -i 's/sha512sum: 498cccba921c019d4526a2a562bd2d9c8efba709ab760fa9d38bd8de1efeefc8e499c9249af9571aa28a1e371e6c928d5175fa70d5d7addcf3dd388caeff1a45/sha512sum: 29d8cf6991272eea20499d757252b07deba824eb778368192f9ab88b215a0dafa584e83422dac08feeb43ddce65f485557ad66210f005a81ab95fb53b7d8d424/g' charts/tinkerbell/stack/values.yaml

			#Update the user provided ip's and proxy's in stack/values.yaml file before starting the tinker bell stack.
			update_load_balancer_ip_and_proxy_settings
			sleep 2
			cd $SCRIPT_DIR/charts/tinkerbell
			#execute the helm command
			helm dependency build stack/
			sleep 2
			trusted_proxies=$(echo $(kubectl get nodes -o jsonpath='{.items[*].spec.podCIDR}' | tr ' ' ','))
			#Some times trusted_proxies variable getting empty value so trying it for few more times.
			if [ -z $trusted_proxies ]; then
				count=0
				while [ $count -le 5 ]; do
					sleep 1
					trusted_proxies=$(echo $(kubectl get nodes -o jsonpath='{.items[*].spec.podCIDR}' | tr ' ' ','))
					if [ ! -z $trusted_proxies ]; then
						break
					else
						count=$((count + 1))
					fi
				done
			fi
			if [ -z $trusted_proxies ]; then
				echo "Not able to get the trusted_proxies ip address , please re run the script once again"
				exit 0
			fi
			echo "Tinkerbell stack installation is going on please wait for few minutes for completion!!!"

			helm install stack-release stack/ --create-namespace --namespace tink-system --wait --set "boots.trustedProxies=${trusted_proxies}" --set "hegel.trustedProxies=${trusted_proxies}" >/dev/null 2>&1

			if [ $? -ne 0 ]; then

				#check if the load balancer ip in pending state for the boots and tink-stack external ip , if yes apply the work around
				lb_ip_for_boots=$(kubectl get all --all-namespaces | grep -i "service/boots" | awk '{print $5}')
				lb_ip_for_tink_stack=$(kubectl get all --all-namespaces | grep -i "service/tink-stack" | awk '{print $5}')

				if [ "$b_ip_for_boots" = "<pending>" ] || [ "$lb_ip_for_tink_stack" = "<pending>" ]; then
					work_around_for_rke2_lb_ip_pending
				fi
			else
				echo "Tinkser bell setup success"
				echo "update_load_balancer_ip_and_proxy_settings done" >>$SCRIPT_DIR/$SETUP_STATUS_FILENAME
				echo "install_tinkerbell_stack  done" >>$SCRIPT_DIR/$SETUP_STATUS_FILENAME
			fi
			cd -
		fi

	fi
}
#Update the loadbalancer ip in values.yaml file and other required Proxy settings for running the Tinkerbell
function update_load_balancer_ip_and_proxy_settings() {
	if grep -q "update_load_balancer_ip_and_proxy_settings done" $SCRIPT_DIR/$SETUP_STATUS_FILENAME; then
		echo "Skipping update_load_balancer_ip"
	else
		sed -i "s/loadBalancerIP:.*/loadBalancerIP: $load_balancer_ip/g" charts/tinkerbell/stack/values.yaml

		sed -i "s/remoteIp:.*/remoteIp: $load_balancer_ip/g" charts/tinkerbell/stack/values.yaml

		sed -i "s/\bip:.*\b/ip: $load_balancer_ip/gI" charts/tinkerbell/stack/values.yaml

		sed -i '/name: dhcp-relay/{n; s/enabled: true/enabled: false/}' charts/tinkerbell/stack/values.yaml

		sed -e '/sourceInterface:/ s/^#*/#/' -i charts/tinkerbell/stack/values.yaml
		sed -i 's/additionlKernelArgs: .*$/additionlKernelArgs: ["console=ttyS0,115200"]/' charts/tinkerbell/boots/values.yaml
		#update the host network to true in boots/values.yaml file
		#	sed -i 's/hostNetwork: false/hostNetwork: true/g'  charts/tinkerbell/boots/values.yaml

		#add the proxy settings in charts/tinkerbell/stack/values.yaml

		http_proxy1=$(sudo cat /etc/environment | grep http_proxy | awk -F "=" '{print $2}')
		https_proxy1=$(sudo cat /etc/environment | grep https_proxy | awk -F "=" '{print $2}')
		no_proxy1=$(sudo cat /etc/environment | grep no_proxy | awk -F "=" '{print $2}')

		sed -i 's/^hegel:/\ \ additionlKernelArgs:\n&/' charts/tinkerbell/stack/values.yaml
		sed -i "/additionlKernelArgs:/a \ \ \ \ -  \"http_proxy=$http_proxy1\"" charts/tinkerbell/stack/values.yaml
		sed -i "/http_proxy/a \ \ \ \ -  \"https_proxy=$https_proxy1\"" charts/tinkerbell/stack/values.yaml
		sed -i "/https_proxy/a \ \ \ \ -  \"no_proxy=$no_proxy1\"" charts/tinkerbell/stack/values.yaml
		sed -i "/no_proxy/a \ \ \ \ -  \"HTTP_PROXY=$http_proxy1\"" charts/tinkerbell/stack/values.yaml
		sed -i "/HTTP_PROXY/a \ \ \ \ -  \"HTTPS_PROXY=$https_proxy1\"" charts/tinkerbell/stack/values.yaml
		sed -i "/HTTPS_PROXY/a \ \ \ \ -  \"NO_PROXY=$no_proxy1\"" charts/tinkerbell/stack/values.yaml
		sed -i "/NO_PROXY/a \ \ \ \ -  \"load_balancer=$load_balancer_ip\"" charts/tinkerbell/stack/values.yaml

		sed -i "/load_balancer/a \ \ \ \ -  \"console=ttyS0,115200\"" charts/tinkerbell/stack/values.yaml
		sed -i "/console/a \ \ \ \ -  \"insecure_registries=$pd_host_ip:5015\"" charts/tinkerbell/stack/values.yaml

		#add the proxy settings in stack/templates/nginx.yaml

		sed -i '/image: {{ .Values.stack.hook.image }}/,/command: ["/bin/bash", "-xeuc"]/a \ \ \ \ \ \ \ \ env:' charts/tinkerbell/stack/templates/nginx.yaml
		sed -i '/image: {{ .Values.stack.hook.image }}/{n;d}' charts/tinkerbell/stack/templates/nginx.yaml
		sed -i '/env:/a \ \ \ \ \ \ \ \ \ \ - name: http_proxy' charts/tinkerbell/stack/templates/nginx.yaml
		sed -i "/name: http_proxy/a \ \ \ \ \ \ \ \ \ \ \ \ value: $http_proxy1" charts/tinkerbell/stack/templates/nginx.yaml
		sed -i '/value: http:\/\/*/a \ \ \ \ \ \ \ \ \ \ - name: https_proxy' charts/tinkerbell/stack/templates/nginx.yaml
		sed -i "/name: https_proxy/a \ \ \ \ \ \ \ \ \ \ \ \ value: $https_proxy1" charts/tinkerbell/stack/templates/nginx.yaml
		sed -i '/value:.*912/a \ \ \ \ \ \ \ \ \ \ - name: no_proxy' charts/tinkerbell/stack/templates/nginx.yaml
		sed -i "/name: no_proxy/a \ \ \ \ \ \ \ \ \ \ \ \ value: $no_proxy1  \n \ \ \ \ \ \ \ \ \ - name: HTTP_PROXY \n \ \ \ \ \ \ \  \ \ \ value: $http_proxy1 \n \ \ \ \ \ \ \ \ \ - name: HTTPS_PROXY \n \ \  \ \ \ \ \ \ \ \ value: $https_proxy1 \n \ \ \ \ \ \ \ \ \ - name: NO_PROXY \n \ \ \ \ \ \ \ \ \ \ \ value: $no_proxy1" charts/tinkerbell/stack/templates/nginx.yaml
		sed -i 's/apt-get update/echo \"nameserver 10.248.2.1\" \>>\ \/etc\/resolv.conf \n \ \ \ \ \ \ \ \ \ echo \"nameserver 172.30.90.4"\ \>>\ \/etc\/resolv.conf \n \ \ \ \ \ \ \ \ \ echo \"nameserver 10.223.45.36"\ \>>\ \/etc\/resolv.conf\n&/g' charts/tinkerbell/stack/templates/nginx.yaml
		sed -i '/echo \"nameserver 10.223.45.36\"/{n;d}' charts/tinkerbell/stack/templates/nginx.yaml

		sed -i '/echo \"nameserver 10.223.45.36\"/a \ \ \ \ \ \ \ \ \ \ apt-get update' charts/tinkerbell/stack/templates/nginx.yaml

		echo "====LOAD BALANCER IP SETUP DONE======="
		#echo "update_load_balancer_ip_and_proxy_settings done" >> $SCRIPT_DIR/$SETUP_STATUS_FILENAME
	fi
}

#Work around for the load balancer ip in pending state
function work_around_for_rke2_lb_ip_pending() {

	#Stop the rke2 service
	sudo systemctl stop rke2-server.service >/dev/null 2>&1
	if [ $? -eq 0 ]; then
		sleep 2
		sudo systemctl start rke2-server.service
		sleep 4
	fi

	if [ ! -f /etc/rancher/rke2/config.yaml ] || [ ! -s /etc/rancher/rke2/config.yaml ]; then
		cat <<EOF | sudo tee /etc/rancher/rke2/config.yaml
tls-san:
- $load_balancer_ip 
disable: rke2-ingress-nginx
cni:
- cilium
EOF
	fi
	export VIP=$load_balancer_ip
	export TAG=v0.5.7
	export INTERFACE=$pub_inerface_name
	export CONTAINER_RUNTIME_ENDPOINT=unix:///run/k3s/containerd/containerd.sock
	export CONTAINERD_ADDRESS=/run/k3s/containerd/containerd.sock
	export PATH=/var/lib/rancher/rke2/bin:$PATH
	sudo cp /var/lib/rancher/rke2/bin/crictl /usr/local/bin/
	export KUBECONFIG=/home/$USER/.kube/config

	sudo bash -c 'curl -s https://kube-vip.io/manifests/rbac.yaml >  /var/lib/rancher/rke2/server/manifests/kube-vip-rbac.yaml'
	sleep 1

	sudo su -c "export CONTAINER_RUNTIME_ENDPOINT=unix:///run/k3s/containerd/containerd.sock && crictl pull docker.io/plndr/kube-vip:$TAG"

	sed -e '/lbClass:.*/ s/^#*/#/' -i $SCRIPT_DIR/charts/tinkerbell/stack/values.yaml
	sed -i '/kubevip:/{n; s/enabled: true/enabled: false/}' $SCRIPT_DIR/charts/tinkerbell/stack/values.yaml

	# removes vip container and snapshot already present in machine
    sudo su -c "export CONTAINERD_ADDRESS=/run/k3s/containerd/containerd.sock && ctr -n k8s.io container list | grep vip"
    if [ $? -eq 0 ]; then
       sudo su -c "export CONTAINERD_ADDRESS=/run/k3s/containerd/containerd.sock && ctr -n k8s.io container delete vip"
    fi
    sudo su -c "export CONTAINERD_ADDRESS=/run/k3s/containerd/containerd.sock && ctr -n k8s.io snapshot list | grep vip"
    if [ $? -eq 0 ]; then
       sudo su -c "export CONTAINERD_ADDRESS=/run/k3s/containerd/containerd.sock && ctr -n k8s.io snapshot delete vip"
    fi

	sudo su -c "export CONTAINERD_ADDRESS=/run/k3s/containerd/containerd.sock && ctr --namespace k8s.io run --rm --net-host docker.io/plndr/kube-vip:$TAG vip /kube-vip manifest daemonset \
                              --arp \
                              --interface $INTERFACE \
                              --address $VIP \
                              --controlplane \
                              --leaderElection \
                              --taint \
                              --services \
                              --inCluster |  tee /var/lib/rancher/rke2/server/manifests/kube-vip.yaml"
	sleep 1
	helm uninstall stack-release --namespace tink-system >/dev/null 2>&1
	cd $SCRIPT_DIR/charts/tinkerbell
	helm dependency build stack/

	sleep 1
	trusted_proxies=$(kubectl get nodes -o jsonpath='{.items[*].spec.podCIDR}' | tr ' ' ',')

	if [ -z $trusted_proxies ]; then
		sleep 2
		trusted_proxies=$(kubectl get nodes -o jsonpath='{.items[*].spec.podCIDR}' | tr ' ' ',')
	fi

	helm install stack-release stack/ --create-namespace --namespace tink-system --wait --set "boots.trustedProxies=${trusted_proxies}" --set "hegel.trustedProxies=${trusted_proxies}"

	if [ $? -eq 0 ]; then
		echo "Done with work around , please check the load balancer ip now"
		echo "update_load_balancer_ip_and_proxy_settings done" >>$SCRIPT_DIR/$SETUP_STATUS_FILENAME
		echo "install_tinkerbell_stack  done" >>$SCRIPT_DIR/$SETUP_STATUS_FILENAME
	else
		echo "Looks ran into some problem please check the logs and re run the workaround"
	fi
	sudo chmod 755 /etc/rancher/rke2/rke2.yaml
	kubectl get all --all-namespaces

}
function instalation_status() {
	status=$(cat $SCRIPT_DIR/$SETUP_STATUS_FILENAME | grep -c done)

	if [[ $status == 9 ]]; then
		echo "Setup completed Successfully"
		echo "Setup completed Successfully" >>$SCRIPT_DIR/$SETUP_STATUS_FILENAME
	else
		echo "Looks few things are not installed properly please check $SCRIPT_DIR/$SETUP_STATUS_FILENAME and re run the setup_tinkerbell_stack.sh "
		echo "Looks few things are not installed properly please check $SCRIPT_DIR/$SETUP_STATUS_FILENAME and re run the setup_tinkerbell_stack.sh " >>$SCRIPT_DIR/$SETUP_STATUS_FILENAME
	fi

}

#Intsall the tinkerbell stack with rke2 cluster
function setup_tinsk_stack() {
	check_pre_condition

	setup_system_proxy

	get_system_ip_details

	update_system_packages

	install_docker_services

	install_kubectl_service

	set_rk2_proxy

	create_rk2e_services

	install_helm_service

	install_tinkerbell_stack
}

###MAIN####

setup_tinsk_stack
echo "Done with the installation , Please check  .tinkerbell_setup_status file for full details"
