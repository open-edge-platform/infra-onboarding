#!/bin/sh

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

set -x

##global variables#####
os_disk=""
part_number=""
data_part_number=""
data_partition_disk=""
rootfs_partition_disk=""
secondary_rootfs_disk_size=3
######################

#lvm creation on disk
create_lvm_partition(){
blk_device_count=$1
shift
lvm_disks="$@"

#if one disk found and it has rootfs
if [ "$blk_device_count" -eq "1" ];then
    echo "starting the LVM creation for the disk volume ${lvm_disks}"
    lvm_part=$(parted -ms ${lvm_disks}  print | tail -n 1 | awk -F: '{print $1}')
    disks="${lvm_disks}${part_number}${lvm_part}"

#more than one disk found
else
    set -- $lvm_disks
    disks=""
    while [ "$1" ]; do
        disk="/dev/$1"
    	echo "starting the LVM creation for the disk volume $disk"
        parted -s "$disk" mklabel gpt mkpart primary 0% 100%
        partprobe
        fdisk -l "$disk"
	sync
	if echo "$disk" | grep -q "nvme"; then
	    part_number="p"
	else
	    part_number=""
	fi
	if [ -z "$disks" ]; then
             disks="${disk}${part_number}1"
	else
             disks="$disks ${disk}${part_number}1"
	fi
    shift
    done
fi
#wipse the crypt luck offset if its created during FDE enabled case
#otherwise LVM creation will fail
set -- $disks
while [ "$1" ];do
    wipefs -o 0 "$1"
    shift
done

#remove previously created lvm if exist
vgs=$(vgs --noheadings -o vg_name)
#remove trailing and leading spaces
vgs=$(echo "$vgs" | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')
if [ -n "$vgs" ]; then
    vgremove -f "$vgs"
    echo "successfully deleted the previous lvm"
fi

#remove previously created pv if exist
pvs=$(pvs --noheadings -o pv_name)
pvs=$(echo "$pvs" | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')
if [ -n "$pvs" ]; then
    for pv in ${pvs}; do
        pvremove -f "$pv"
        echo "successfully deleted the previous pvs"
    done
fi
#pv create
set -- $disks
while [ "$1" ];do
        if echo "y" | pvcreate "$1"; then
            echo "Successfuly done pvcreate"
        else
            echo "Failure in pvcreate"
            exit 1
        fi
	shift
done

#vgcreate
if echo "y" | vgcreate lvmvg "$disks"; then
    echo "Successfuly done vgcreate"
else
    echo "Failure in vgcreate"
    exit 1
fi

vgscan
vgchange -ay

if vgchange -ay; then
    echo "Successfuly created the logical volume group"
else
    echo "Failure in creating the logical volume group"
    exit 1
fi
}
#swap partition creation
create_swap_partition(){
disk=$1
SWAP_PART_SIZE_START=$2
SWAP_PART_SIZE_END=$3
data_part_num=$4
swap_part_number=$((data_part_num+2))
parted "${disk}" --script mkpart primary linux-swap "${SWAP_PART_SIZE_START}GB"  "${SWAP_PART_SIZE_END}GB"
#get the partition number
if echo "$disk" | grep -q "nvme"; then
    swap_part="p${swap_part_number}"
else
    swap_part="${swap_part_number}"
fi
partprobe "${disk}"
mkswap "/dev/${os_disk}${swap_part}"
swapon "/dev/${os_disk}${swap_part}"

#get the UUID
uuid=$(blkid | grep swap | grep primary | awk '{print $2}'|awk -F= '{print $2}'|tr -d '"')
if [ -z "$uuid" ]; then
    echo "Faild to create the swap partiton!!!!"
    exit 1
else
    #add entry for swap partition in fstab
   
    #Before mouting the file system check if the file system exist or not from user space, if not wait for few seconds
    count=0
    partprobe "${disk}" 
    while [ ! -b "$rootfs_partition_disk" ] && [ "$count" -le 9 ]; do
	    sleep 1 && count=$((count+1))
    done
    if [ "$count" -ge 10 ]; then 
        echo "Faild to mount the root file system for for swap entry update $disk"
        exit 1
    fi
    mount $rootfs_partition_disk /mnt
    mount --bind /dev /mnt/dev
    mount --bind /dev/pts /mnt/dev/pts
    mount --bind /proc /mnt/proc
    mount --bind /sys /mnt/sys
    echo "UUID=$uuid swap swap default 0 2" >> /mnt/etc/fstab
    status=$(cat "/mnt/etc/fstab" | grep -c "swap")
    if [ "$status" -ge 1 ]; then
        echo "Successfuly created the swap partition for the disk $disk"
    else
	echo "Faild to update swap partition in /etc/fstab for the disk $disk"
	exit 1
    fi	
    #unmount the partitions
    for mount in $(mount | grep '/mnt' | awk '{print $3}' | sort -nr); do
        umount "$mount"
    done
fi
}
#disk partition for SWAP and LVM
partition_disk(){
ram_size=$1
disk_size=$2

disk="/dev/$os_disk"

#get the number of devices attached to system ignoreing USB/Virtual/Removabale disks
blk_devices=$(lsblk -o NAME,TYPE,SIZE,RM | grep -i disk | awk '$1 ~ /sd*|nvme*/ {if ($3 !="0B" && $4 ==0)  {print $1}}')
set -- $blk_devices
blk_disk_count=$#
final_disk_list=""
for disk_name in ${blk_devices}
do
    #skip for rootfs disk
    if echo "$disk_name" | grep -q "$os_disk"; then
        continue;
    else 
        if [ -z "$final_disk_list" ]; then
	    final_disk_list="$disk_name"
	else
            final_disk_list="$final_disk_list $disk_name"
	fi
    fi
done
if [ "$blk_disk_count" -eq 1 ]; then
    #create the SAWP size as square root of ram size
    swap_size=$(echo "scale=0; sqrt($ram_size)" | bc)
else
    #create the swap size as half of RAM size
    swap_size=$((ram_size/2))
    #cap the swap_size to 128GB
    if [ "$swap_size" -gt 128 ]; then
        swap_size=128
    fi
fi

#make sure swap size should not exceed the total disk size
if [ "$swap_size" -ge "$disk_size" ]; then
    echo "Looks the Disk size is very Minimal and can't proceed with partition!!!!"
    exit 1
fi
#get the partition for the rootfs for side B for A/B upgrades
secondary_rootfs_disk_num=$((data_part_number+1))
if echo "$disk" | grep -q "nvme"; then
    secondary_rootfs_disk="p${secondary_rootfs_disk_num}"
else
    secondary_rootfs_disk="${secondary_rootfs_disk_num}"
fi

#expand the tiber_persistent partition on rootfs disk

if [ "$blk_disk_count" -eq 1 ]; then
    #expand the tiber_persistent partition max to 100GB if only one disk
    new_disk_partition_size="100"
    #secondary rootfs partitions for A/B day2 upgrades
    secondary_rootfs_disk_end=$((new_disk_partition_size+secondary_rootfs_disk_size))

    parted ---pretend-input-tty "${disk}" \
        resizepart "$data_part_number" "${new_disk_partition_size}GB" \
        mkpart primary ext4 "${new_disk_partition_size}GB" "${secondary_rootfs_disk_end}GB"
    if [ $? -ne 0 ]; then
        echo "Partition creation failed for the disk ${disk} failed"
        exit 1
    else
        echo "Partition creation for the disk ${disk} Successful!!"
    fi
    partprobe "${disk}"
else
    #more than one disk detected expand the tiber_persistent partition to max-swap  partition

    #get the last partition end point
    data_part_end=$(parted -m $disk unit GB print | grep "^$data_part_number" | cut -d: -f3 | sed 's/GB//')
    if echo "$data_part_end" | grep -qE '^[0-9]+\.[0-9]+$'; then
        data_part_end=$(printf "%.0f" "$data_part_end")
    fi
    #add data_part_end secondary_rootfs disk size and swap_size to get toatl size in use
    total_size_inuse=$(echo "$data_part_end + $swap_size + $secondary_rootfs_disk_size" | bc)
    #calculate the size for expanding the data partition
    data_part_end_size=$(echo "$disk_size - $total_size_inuse" | bc)
    #secondary rootfs partitions for A/B day2 upgrades
    secondary_rootfs_disk_end=$((data_part_end_size+secondary_rootfs_disk_size))
    parted ---pretend-input-tty "${disk}" \
        resizepart "$data_part_number" "${data_part_end_size}GB" \
       	mkpart primary ext4 "${data_part_end_size}GB" "${secondary_rootfs_disk_end}GB"
    if [ $? -ne 0 ]; then
        echo "Partition resize for the disk ${disk} failed"
        exit 1
    else
        echo "Partition resize for the disk ${disk} Successful!!"
    fi
    partprobe "${disk}"

fi

#get the end size of the last partition from the  disk
last_partition_end=$(parted -ms $disk  print | tail -n 1 | awk -F: '{print $3}' | sed 's/GB$//')
if echo "$last_partition_end" | grep -qE '^[0-9]+\.[0-9]+$'; then
        last_partition_end=$(printf "%.0f" "$last_partition_end")
fi
swap_partition_size_end=$((last_partition_end+swap_size))

#create SWAP
create_swap_partition "${disk}" "${last_partition_end}" "${swap_partition_size_end}" "${data_part_number}"

#create LVM
#If the Number of Disks ditected=1 then create LVM partitions on same disk
if [ "$blk_disk_count" -eq 1 ]; then

    echo "found single disk for LVM creation"
    #create LVM partition
    blk_disk_count=1
    lvm_partition_size="100%"
    swap_partition_size_end=$(parted -ms $disk  print | tail -n 1 | awk -F: '{print $3}' | sed 's/[^0-9]*//g')
    parted "${disk}" --script mkpart primary ext4 "${swap_partition_size_end}GB" $lvm_partition_size
    partprobe "${disk}"

    create_lvm_partition "${blk_disk_count}" "${disk}" 

#if more than 1 disk ditected then create the LVM partition on secondary disks
else
    echo "found more than 1 disk for LVM creation"
    create_lvm_partition  "${blk_disk_count}" "${final_disk_list}" 
fi

#finally expand the data partition using resize2fs
e2fsck -f "$data_partition_disk"
resize2fs "$data_partition_disk"
if [ $? -ne 0 ]; then
    echo "Partition resize for the disk ${disk} failed"
    exit 1
else
    echo "Partition resize for the disk ${disk} Successful!!"
    fi
}

#######@main

echo "--------Starting the SWAP and LVM partition on TiberOS---------"

#get the rootfs partition from the disk

rootfs_partition_disk=$(blkid | grep -i rootfs | grep -i ext4 |  awk -F: '{print $1}')
data_partition_disk=$(blkid | grep -i "tiber_persistent" | grep -i ext4 |  awk -F: '{print $1}')

if echo "$rootfs_partition_disk" | grep -q "nvme"; then
    os_disk=$(echo "$rootfs_partition_disk" | grep -oE 'nvme[0-9]+n[0-9]+' | head -n 1)
    part_number="p"
    data_part_number=$(blkid | grep "tiber_persistent" | awk -F'[/:]' '{print $3}'| awk -F'p' '{print $2}')
else
    os_disk=$(echo "$rootfs_partition_disk" | grep -oE 'sd[a-z]+' | head -n 1)
    part_number=""
    data_part_number=$(blkid | grep "tiber_persistent" | awk -F'[/:]' '{print $3}' | sed 's/[^0-9]*//g')
fi

#check the ram size && decide the sawp size based on it

ram_size=$(free -g | grep -i mem | awk '{ print $2 }')

#get the total rootfs partition disk size

sgdisk -e "/dev/$os_disk"
total_disk_size=$(parted -m "/dev/$os_disk" unit GB print | grep "^/dev" | cut -d: -f2 | sed 's/GB//')
if echo "$total_disk_size" | grep -qE '^[0-9]+\.[0-9]+$'; then
	total_disk_size=$(printf "%.0f" "$total_disk_size")
fi

#partition the disk with swap and LVM

partition_disk "$ram_size" "$total_disk_size"  
