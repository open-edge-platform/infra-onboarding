##################################################
#
# Readme.md file for zero touch onboarding automation
#
#################################################

Before running the script make sure that you reserved the free intel network IP from below tool for load balancer IP and SUT IP
https://ddi.intel.com/menandmice/Login.htm?SSO=1

once you got the intel network IP for load balancer and  sut,  update the sut_onboarding_list.txt file, see below for an example. You can give as as many as SUT’s for providing, just update correct entries under the sut_onboarding_list.txt file

#Example
#SUT_NAME  #MAC_ID           #Load_Balancer_IP   #SUT_IP        #Disk_type      #Image_Type

#SUT1     00:49:fa:07:8d:05   10.199.199.100    10.199.199.101   /dev/neme0n1   prod_bkc/prod_focal/prod_focal-ms/prod_jammy

execute the script ./zero_touch_onboarding_installation.sh

for installation status please check for . onboarding_installation_status  and.

for installation logs refer to onboarding_logs.txt from the same directory.

