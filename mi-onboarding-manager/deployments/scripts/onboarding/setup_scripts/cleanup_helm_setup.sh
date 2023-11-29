#!/bin/bash

#Cleaning up the helm chart
helm uninstall  fdo-db  fdo-mfg fdo-owner fdo-rv

#Cleaning up existing configmaps
kubectl delete cm fdo-mfg-service-env fdo-owner-service-env fdo-rv-service-env

rm -rf /home/$USER/error_log_FDO

#To cleanup the Onboarding-Manager helm chart
helm uninstall onb-mgr
