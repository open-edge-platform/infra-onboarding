# On-boarding

1. Ensure all the FDO images have been built and uploaded.
2. Ensure the FDO setup script has been executed. All the required files are created under /home/\<USER\>/pri-fidoiot/component-samples/demo/
3. Ensure there is no FDO services and MariaDB running in docker container.

## Deploy FDO DB

1. Update the path of required files in fdo-db/values.yaml.
2. Apply the helm command.
    `helm install fdo-db fdo-db/`
3. Wait around 25 seconds to make MariaDB internal database up.
4. If the database has a CrashLoopError and the error mentions the InnoDB is unsupported, please kindly change the default-storage-engine
to other type, for example, **Memory**. The config file of the database can be found in /home/\<USER\>/pri-fidoiot/component-samples/demo/db/custom/config-file.cnf


## Deploy FDO RV

1. Ensure all the required files are created.
2. Ensure FDO DB (MariaDB) is running.
3. Create the configmap for service.env with command below.
    `kubectl create configmap fdo-rv-service-env --from-env-file=<path to FDO RV service.env>`
4. Update the host internal IP and path of required files in fdo-rv/values.yaml.
5. Update the **host.docker.internal** in /home/\<USER\>/pri-fidoiot/component-samples/demo/rv/service.yml to the machine's ip.
6. Apply the helm command.
    `helm install fdo-rv fdo-rv/`

## Deploy FDO MFG

1. Ensure all the required files are created.
2. Ensure FDO DB (MariaDB) is running.
3. Create the configmap for service.env with command below.
    `kubectl create configmap fdo-mfg-service-env --from-env-file=<path to FDO MFG service.env>`
4. Update the host internal IP and path of required files in fdo-mfg/values.yaml.
5. Update the **host.docker.internal** in /home/\<USER\>/pri-fidoiot/component-samples/demo/manufacturer/service.yml to the machine's ip.
6. Apply the helm command.
    `helm install fdo-mfg fdo-mfg/`

## Deploy FDO Owner

1. Ensure all the required files are created.
2. Ensure FDO DB (MariaDB) is running.
3. Create the configmap for service.env with command below.
    `kubectl create configmap fdo-owner-service-env --from-env-file=<path to FDO Owner service.env>`
4. Update the host internal IP and path of required files in fdo-mfg/values.yaml.
5. Update the **host.docker.internal** in /home/\<USER\>/pri-fidoiot/component-samples/demo/owner/service.yml to the machine's ip.
6. Apply the helm command.
    `helm install fdo-owner fdo-owner/`
