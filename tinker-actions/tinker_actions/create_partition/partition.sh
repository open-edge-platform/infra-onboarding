#####################################################################################
# INTEL CONFIDENTIAL                                                                #
# Copyright (C) 2023 Intel Corporation                                              #
# This software and the related documents are Intel copyrighted materials,          #
# and your use of them is governed by the express license under which they          #
# were provided to you ("License"). Unless the License provides otherwise,          #
# you may not use, modify, copy, publish, distribute, disclose or transmit          #
# this software or the related documents without Intel's prior written permission.  #
# This software and the related documents are provided as is, with no express       #
# or implied warranties, other than those that are expressly stated in the License. #
#####################################################################################

#set -x

disk=${BLOCK_DEVICE:-'/dev/sdb'}      # disk name
partition_size=${PARTITION_SZ:-200MB} # size of partition

partition_label=${LABEL:-'CREDS'} # Partition label
detected_disk="False"

set_pxe_default() {
    ######## make PXE the first boot option possible.
    pxe_boot_number=$(efibootmgr | grep -i "Bootcurrent" | awk '{print $2}')

    boot_order=$(efibootmgr | grep -i "Bootorder" | awk '{print $2}')

    remove_pxe=$(sed "s/$pxe_boot_number//g" <<<$boot_order)
    remove_pxe=$(sed "s/,,/,/g" <<<$remove_pxe)

    final_bootorder=$pxe_boot_number","$remove_pxe
    final_bootorder=$(sed "s/,,/,/g" <<<$final_bootorder)
    efibootmgr --bootorder $final_bootorder
    echo "Made PXE the first in the boot order"
    #until we change it in the store-alpine run.
    #needed because there is a bug in the reboot sequence.
    #basically before we update tink the restart happens.
    #This way we ensure that boot to pxe until we are done.
}

# Check if 'parted' command is available
if ! command -v parted >/dev/null 2>&1; then
    echo "Error: 'parted' command not found. Please install 'parted' and try again."
    exit 1
fi

# check for the correct boot partition.
blk_devices=$(parted -l | grep -ie "/dev/nvme" -ie "/dev/sd." -m3 | sort | awk '{ print substr($2, 1, length($2)-1) }')
for blk in $blk_devices; do
    : $(parted -s $blk print | grep -ie "bios_grub" -ie "boot, esp")
    if [ $? -eq 0 ]; then
        echo "Block device detected with grub $blk"
        disk=$blk
        detected_disk="True"
        break
    fi
done

if [ "$detected_disk" == "False" ]; then
    selected_disk=$(parted -l | grep -i "/dev/" | awk '{ print substr($2, 1, length($2)-1) }' | grep -i "nvme.n." -m 1)
    if [ $? -eq 0 ]; then
        #seleted NVME
        disk=$selected_disk
    else
        #selected sda/sdb/sdc/sdd/....
        selected_disk=$(parted -l | grep -i "/dev/" | awk '{ print substr($2, 1, length($2)-1) }' | grep -i "sd." -m 1)
        if [ $? -eq 0 ]; then
            disk=$selected_disk
            break
        fi
    fi
fi

#echo "Block device detected $disk"
printf "Block device detected $disk "

#total paritions
line_num=$(parted -s "$disk" print | awk '$1 == "Number" { print NR }')
partition_num=$(parted -s "$disk" print | awk 'NR > $line_num { print $1}')
for part in $partition_num; do
    echo "partition in $disk $part will be deleted"
    rm_part=$(parted -s "$disk" rm "$part")
done

#pxe_boot_num=$(efibootmgr | grep -i "BootCurrent" | awk ' { print $2 }')
#set pxe as the next boot
#efibootmgr --bootnext $pxe_boot_num
#echo "Completed setting PXE as next boot"
set_pxe_default

#check if TPM present of device Else FDO will work on client-sdk
ret=$(ls /dev/tpm*)
if [ $? == 0 ]; then
    #use CLIENT_SDK_TPM
    printf "FDO client - CLIENT_SDK_TPM \n"
else
    #use CLIENT_SDK_TPM
    printf "FDO client - CLIENT_SDK \n"
fi
