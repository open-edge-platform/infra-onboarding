#!/bin/sh


#read /proc/cmdline and extract string after worker_id
for i in $(cat $(pwd)/cmdline); do
    if [ "${i#worker_id=}" != "$i" ]; then
        worker_id=${i#worker_id=}
        break
    fi
done
echo "Worker ID: ${worker_id}"

SLEEP_TIME=3
NUMBER_OF_RETRIES=10

#check if IP address is assigned to interface matching MAC address with worker_id
for iface in $(ls /sys/class/net); do
    mac_address=$(cat /sys/class/net/$iface/address)
    for attempt in $(seq 1 $NUMBER_OF_RETRIES); do
        if [ "$mac_address" = "$worker_id" ]; then
            ip_address=$(ip addr show $iface | grep -w inet | awk '{print $2}' | cut -d/ -f1)
            if [ -n "$ip_address" ]; then
                echo "IP Address $ip_address is assigned to interface $iface with MAC $mac_address"
                exit 0
            else
                echo "Attempt $attempt/$NUMBER_OF_RETRIES: No IP address assigned to interface $iface with MAC $mac_address yet"
                sleep $SLEEP_TIME
            fi
        fi
    done
done

echo "No interface found with MAC address matching worker_id: $worker_id"
exit 1
