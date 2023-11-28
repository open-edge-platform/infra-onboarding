Installation of FDO Services as Helm Charts

#Make sure the FDO services are not running as Docker containers( Verfiy with command $docker ps ). 
Helm charts for FDO services that will be deployed are stored at 
infrastructure.edge.iaas.platform/helm/edge-iaas-platform/platform-director/on-boarding 

Steps: 
1).Update the config file with required details before proceeding with the helm setup script

2).Execute the script with below command ./helm_setup_script.sh

chmod +x helm_setup_script.sh # If permission is denied for executing the script

This script will update the values.yaml files with pd_host_ip & location of secrets and deploy the helm charts in sequence.
Script will automatically cleanup the existing FDO service pods and Configmaps if exists already. If needed to cleanup manually, please execute cleanup_helm_setup.sh

Please refer to $HOME/error_log_FDO if the script fails and make sure the FDO pods are in "Running" status($kubectl get pods).
