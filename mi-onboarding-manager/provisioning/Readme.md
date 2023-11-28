##################################################
#
# Readme.md file for Tinkerbell with intel network
#
#################################################

Before running the script make sure that you reserved free intel network IP from IT Team (same subnet as host IP) for Load balancer IP.

Reserving the Edge node IP (SUT) for downloading the ipxe.cfg file using below URL.
https://ddi.intel.com/menandmice/Login.htm?SSO=1

please look at Intel_network_IP_Reservation_Procedure.doc for detailed steps.

update config file with required details before executing the script.

execute the script ./setup_tinkerbell_stack_with_intel_network.sh

first time it will set the env variables and ask for the reboot, please reboot the system for environment variables to apply for smooth download

once system comes up rerun the script ./setup_tinkerbell_stack_with_intel_network.sh

for installation status please check for .tinkerbell_setup_status

NOTE: if you want to re-execute any fuction like rke2/tinkerbell/docker simply just delete the done state of a particular function
name from .tinkerbell_setup_status and rerun the script.



