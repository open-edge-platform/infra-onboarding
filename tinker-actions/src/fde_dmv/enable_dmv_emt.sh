#!/bin/bash

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

#####################################################################################
####
####
#set -x
# COMPLETE_DMVERITY set to true default
COMPLETE_DMVERITY=true

# Test flag for enabling DM-verity on B part aswell.
TEST_ENABLE_DM_ON_ROOTFSB=false

# Test flag for only partition
TEST_ON_ONLY_ONE_PART=false

# Set PARTITION_MODE to either EN (Edge Node) or VEN (Virtual Edge Node)
DMV_IN_VEN=true # Defaelt to EN if not set

####
####
#####################################################################################
# Patition information
boot_partition=1
rootfs_partition=2
emt_persistent_partition=3

#efi_partition=15

root_hashmap_a_partition=4
root_hashmap_b_partition=5
rootfs_b_partition=6
roothash_partition=7

swap_partition=8
singlehdd_lvm_partition=9


# DEST_DISK set from the template.yaml file as an environment variable.

#####################################################################################

# Size in MBs for EN
# Partitions in %
swap_space_start=91

# Size in MBs
tep_size=14336
reserved_size=5120
boot_size=5120600
bare_min_rootfs_size=25
rootfs_size=3584
rootfs_hashmap_size=100
rootfs_roothash_size=50

#####################################################################################
#Global var which is updated
single_hdd=-1
check_all_disks=1
#####################################################################################
#####################################################################################
set_ven_partitions() {
    # Size in MBs for VEN
    rootfs_size=2048
    rootfs_hashmap_size=100
    rootfs_roothash_size=50
    swap_size=4096
    reserved_size=3072
}

#####################################################################################
fix_partition_suffix() {
    part_variable=''
    ret=$(grep -i "nvme" <<< "$DEST_DISK")
    if [ $? == 0 ]
    then
	part_variable="p"
    fi

    echo $part_variable
}

#####################################################################################
get_partition_suffix() {
    part_variable=''
    ret=$(grep -i "nvme" <<< "$1")
    if [ $? == 0 ]
    then
	part_variable="p"
    fi

    echo $part_variable
}

#####################################################################################
check_return_value() {
    if [ $1 -ne 0 ]
    then
	echo "$2"
	exit 1
    fi
}

#####################################################################################
get_dest_disk()
{
    disk_device=""

    list_block_devices=($(lsblk -o NAME,TYPE,SIZE,RM | grep -i disk | awk '$1 ~ /sd*|nvme*/ {if ($3 !="0B" && $4 ==0)  {print $1}}'))
    for block_dev in ${list_block_devices[@]};
    do
	#if there were any problems when the ubuntu was streamed.
	printf 'OK\n'  | parted ---pretend-input-tty -m  "/dev/$block_dev" p
	printf 'Fix\n' | parted ---pretend-input-tty -m  "/dev/$block_dev" p

	parted "/dev/$block_dev" p | grep -i boot
	if [ $? -ne 0 ];
	then
	   continue
	fi

	disk_device="/dev/$block_dev"
    done

    if [[ -z $disk_device ]];
    then
	echo "Failed to get the disk device: Most likely no OS was installed"
	exit 1
    fi

    export DEST_DISK=$disk_device
    echo "DEST_DISK set as $DEST_DISK"
}

#####################################################################################
# set the single_hdd var to 0 if this is a single HDD else it will keep it unchanged at -1
is_single_hdd() {
    ret=-1
    # list_block_devices=($(lsblk -o NAME,TYPE | grep -i disk  | awk  '$1 ~ /sd*|nvme*/ {print $1}'))
    ## $3 represents the block device size. if 0 omit
    ## $4 is set to 1 if the device is removable
    list_block_devices=($(lsblk -o NAME,TYPE,SIZE,RM | grep -i disk | awk '$1 ~ /sd*|nvme*/ {if ($3 !="0B" && $4 ==0)  {print $1}}'))

    count=${#list_block_devices[@]}

    if [ $count -eq 0 ];
    then
	echo "No valid block devices found."
	exit 1
    fi

    if [ $count -eq 1 ];
    then
	# send a 0 if there is only one HDD
	single_hdd=0
	echo "Single Disk selected"
    fi

}

#####################################################################################
# pass value to parted via sector number insted of start size in MB.

convert_mb_to_sectors() {
    local size_in_mb=$1
    local end_sector=$2
    local sectors=$(( size_in_mb * 2048 - end_sector ))
    echo $sectors
}

#####################################################################################

#####################################################################################
make_partition() {

    #ram_size
    ram_size=$(free -g | grep -i mem | awk '{ print $2 }' )
    swap_size=$(( ram_size / 2 ))


    # limit swap size to 128GB
    if [[ $swap_size -gt 128 ]];
    then
	swap_size=128
    fi


    if [ $single_hdd -eq 0 ];
    then
	#
	# limit swap size to sqrt of ramsize link https://help.ubuntu.com/community/SwapFaq
	#
	# this is to reconcile the requirement where we have a upper limit of 100GB for
	# all partitions apart from lvm we cant risk exceeding the swap size.
	swap_size=$(echo "$ram_size" | awk '{printf ("%.0f\n", sqrt($1))}')
    fi

    swap_size=$(( swap_size * 1024 ))

    total_size_disk=$(fdisk -l ${DEST_DISK} | grep -i ${DEST_DISK} | head -1 |  awk '/GiB/{ print int($3)*1024} /TiB/{ print int($3)*1024*1024}')

    # For single HDD reduce the size to 100 and fit everything inside it
    if [ $single_hdd -eq 0 ];
    then
	total_size_disk=$(( 100 * 1024 ))
    fi

    #####
    if $COMPLETE_DMVERITY;
    then
	swap_start=$(( total_size_disk - swap_size ))

	roothash_start=$(( swap_start - rootfs_roothash_size ))
	rootfs_b_start=$(( roothash_start - rootfs_size ))
	root_hashmap_b_start=$(( rootfs_b_start - rootfs_hashmap_size ))
	root_hashmap_a_start=$(( root_hashmap_b_start - rootfs_hashmap_size ))

	emt_persistent_end=$root_hashmap_a_start
    else
	reserved_start=$(( total_size_disk - reserved_size ))
	tep_start=$(( reserved_start - tep_size ))
	swap_start=$(( tep_start - swap_size ))

	rootfs_b_start=$(( swap_start - rootfs_size ))

	emt_persistent_end=$rootfs_b_start
    fi
    #####

    #save this size of emt persistent before partition
    suffix=$(fix_partition_suffix)
    emt_persistent_dd_count=$(fdisk -l ${DEST_DISK} | grep "${DEST_DISK}${suffix}${emt_persistent_partition}" | awk '{print int( ($4/2048/4) + 0.999999) }')
    export emt_persistent_dd_count
    #####
    # logging needed to understand the block splits
    echo "DEST_DISK ${DEST_DISK}"
    echo "rootfs_partition     $rootfs_partition         rootfs_end           ${rootfs_end}MB"
    echo "root_hashmap_a_start ${root_hashmap_a_start}MB root_hashmap_b_start ${root_hashmap_b_start}MB"
    echo "root_hashmap_b_start ${root_hashmap_b_start}MB rootfs_b_start       ${rootfs_b_start}MB"
    echo "rootfs_b_start       ${rootfs_b_start}MB       roothash_start       ${roothash_start}MB"
    echo "roothash_start       ${roothash_start}MB       swap_start           ${swap_start}MB"
    echo "swap_start          ${swap_start}MB            tep_start            ${total_size_disk}MB"
    #####

    echo "sizes in sectors"
    echo "rootfs_partition     $rootfs_partition         rootfs_end             $(convert_mb_to_sectors ${rootfs_end} 1)"
    echo "root_hashmap_a_start $(convert_mb_to_sectors ${root_hashmap_a_start} 0) root_hashmap_b_start $(convert_mb_to_sectors ${root_hashmap_b_start} 1)"
    echo "root_hashmap_b_start $(convert_mb_to_sectors ${root_hashmap_b_start} 0) rootfs_b_start       $(convert_mb_to_sectors ${rootfs_b_start} 1)"
    echo "rootfs_b_start       $(convert_mb_to_sectors ${rootfs_b_start} 0)       roothash_start       $(convert_mb_to_sectors ${roothash_start} 1)"
    echo "roothash_start       $(convert_mb_to_sectors ${roothash_start} 0)       swap_start           $(convert_mb_to_sectors ${swap_start} 1)"
    echo "swap_start          $(convert_mb_to_sectors ${swap_start} 0)            tep_start            $(convert_mb_to_sectors ${total_size_disk} 1)"
    #####

    if $COMPLETE_DMVERITY;
    then
	#this cmd only resizes parition. if there is an error this should handle it.
	printf 'Fix\n' | parted ---pretend-input-tty ${DEST_DISK} \
	       resizepart $emt_persistent_partition "$(convert_mb_to_sectors "${emt_persistent_end}" 1)"s

	check_return_value $? "Failed to resize emt persistent paritions"

	#this cmd only creates new partitions.
	parted -s ${DEST_DISK} \
	       mkpart hashmap_a ext4  "$(convert_mb_to_sectors "${root_hashmap_a_start}" 0)"s "$(convert_mb_to_sectors "${root_hashmap_b_start}" 1)"s \
	       mkpart hashmap_b ext4  "$(convert_mb_to_sectors "${root_hashmap_b_start}" 0)"s "$(convert_mb_to_sectors "${rootfs_b_start}" 1)"s \
	       mkpart rootfs_b ext4   "$(convert_mb_to_sectors "${rootfs_b_start}" 0)"s       "$(convert_mb_to_sectors "${roothash_start}" 1)"s \
	       mkpart roothash ext4   "$(convert_mb_to_sectors "${roothash_start}" 0)"s       "$(convert_mb_to_sectors "${swap_start}" 1)"s \
	       mkpart swap linux-swap "$(convert_mb_to_sectors "${swap_start}" 0)"s           "$(convert_mb_to_sectors "${total_size_disk}" 1)"s 


	check_return_value $? "Failed to create paritions"
    else
	parted -s ${DEST_DISK} \
	       resizepart $emt_persistent_partition "${emt_persistent_end}MB" \
	       mkpart rootfs_b ext4 "${rootfs_b_start}MB" "${swap_start}MB" \
	       mkpart swap linux-swap "${swap_start}MB" "${tep_start}MB" \
	       mkpart tep ext4 "${tep_start}MB"  "${reserved_start}MB" \
	       mkpart reserved ext4 "${reserved_start}MB"  "${total_size_disk}MB"

	check_return_value $? "Failed to create paritions"
    fi


    if [ $single_hdd -eq 0 ];
    then
	parted -s ${DEST_DISK} \
	       mkpart lvm ext4 "$(convert_mb_to_sectors "${total_size_disk}" 0)"s 100%

	check_return_value $? "Failed to create lvm parition"
    fi


    suffix=$(fix_partition_suffix)

    if $COMPLETE_DMVERITY;
    then
	#roothash partition
	mkfs -t ext4 -L roothash -F "${DEST_DISK}${suffix}${roothash_partition}"
	check_return_value $? "Failed to mkfs roothash part"

	# rootfs for a/B updated
	mkfs -t ext4 -L root_b -F "${DEST_DISK}${suffix}${rootfs_b_partition}"
	check_return_value $? "Failed to mkfs rootfs part"
    fi
}

#####################################################################################
make_partition_ven() {
    # Logic for partitioning when PARTITION_MODE is VEN

    total_size_disk=$(fdisk -l ${DEST_DISK} | grep -i ${DEST_DISK} | head -1 |  awk '/GiB/{ print int($3)*1024} /TiB/{ print int($3)*1024*1024}')
    boot_size=$(lsblk -bno SIZE "${DEST_DISK}1")
    boot_size=$((boot_size / 1024 / 1024))  # Convert bytes to MB

    if [ $total_size_disk -lt $((32 * 1024)) ];
    then
        echo "Error: Total disk size is less than 32GB. Exiting."
        exit 1
    fi

    fixed_size=$((rootfs_size * 2 + rootfs_hashmap_size * 2 + rootfs_roothash_size + swap_size + reserved_size  + boot_size))
    remaining_size=$((total_size_disk - fixed_size))
    if [ "$remaining_size" -le 0 ];
    then
            echo "Error: Not enough disk space"
            exit 1
    fi

    if [ $single_hdd -eq 0 ];
    then
        forty_percent=$((remaining_size * 40 / 100))
        emt_persistent_size=$((forty_percent <= 20480 ? forty_percent : 20480))
        lvm_size=$((remaining_size - emt_persistent_size))
    elif [ "$single_hdd" -ne 0 ];
    then 
        emt_persistent_size=$remaining_size
        lvm_size=0
    fi


    rootfs_end=$(parted -s "${DEST_DISK}" unit s print | awk '/ 2 / {gsub("s", "", $3); print int($3 * 512 / 1024 / 1024)}')

    if [ -z "$rootfs_end" ]; then
        echo "Error: Could not determine end of rootfs_a (sda2)"
        exit 1
    fi

     # Compute start points in MB
    emt_persistent_start_mb=$rootfs_end
    emt_persistent_end_mb=$((emt_persistent_start_mb + emt_persistent_size))
    root_hashmap_a_start=$emt_persistent_end_mb
    root_hashmap_b_start=$((root_hashmap_a_start + rootfs_hashmap_size))
    rootfs_b_start=$((root_hashmap_b_start + rootfs_hashmap_size))
    roothash_start=$((rootfs_b_start + rootfs_size))
    swap_start=$((roothash_start + rootfs_roothash_size))
    reserved_start=$((swap_start + swap_size))
    lvm_start=$((reserved_start + reserved_size))
    lvm_end=$((lvm_start + lvm_size))

    #####
    # logging needed to understand the block splits
    echo "DEST_DISK             ${DEST_DISK}"
    echo "rootfs_partition     $rootfs_partition         rootfs_end           ${rootfs_end}MB"
    echo "emt_persistent_start ${emt_persistent_start_mb}MB emt_persistent_end   ${emt_persistent_end_mb}MB"
    echo "root_hashmap_a_start ${root_hashmap_a_start}MB root_hashmap_b_start ${root_hashmap_b_start}MB"
    echo "root_hashmap_b_start ${root_hashmap_b_start}MB rootfs_b_start       ${rootfs_b_start}MB"
    echo "rootfs_b_start       ${rootfs_b_start}MB       roothash_start       ${roothash_start}MB"
    echo "roothash_start       ${roothash_start}MB       swap_start           ${swap_start}MB"
    echo "swap_start           ${swap_start}MB           reserved_start       ${reserved_start}MB"
    echo "reserved_start       ${reserved_start}MB	 lvm_start	      ${lvm_start}MB"
    echo "lvm_start            ${lvm_start}MB            lvm_end              ${lvm_end}MB"
    #####
    
    echo "sizes in sectors"
    echo "rootfs_partition     $rootfs_partition         rootfs_end             $(convert_mb_to_sectors ${rootfs_end} 1)"
    echo "emt_persistent_start $(convert_mb_to_sectors ${emt_persistent_start_mb} 0) emt_persistent_end   $(convert_mb_to_sectors ${emt_persistent_end_mb} 1)"
    echo "root_hashmap_a_start $(convert_mb_to_sectors ${root_hashmap_a_start} 0) root_hashmap_b_start $(convert_mb_to_sectors ${root_hashmap_b_start} 1)"
    echo "root_hashmap_b_start $(convert_mb_to_sectors ${root_hashmap_b_start} 0) rootfs_b_start       $(convert_mb_to_sectors ${rootfs_b_start} 1)"
    echo "rootfs_b_start       $(convert_mb_to_sectors ${rootfs_b_start} 0)       roothash_start       $(convert_mb_to_sectors ${roothash_start} 1)"
    echo "roothash_start       $(convert_mb_to_sectors ${roothash_start} 0)       swap_start           $(convert_mb_to_sectors ${swap_start} 1)"
    echo "swap_start           $(convert_mb_to_sectors ${swap_start} 0)           reserved_start       $(convert_mb_to_sectors ${reserved_start} 1)"
    echo "reserved_start       $(convert_mb_to_sectors ${reserved_start} 0)       lvm_start            $(convert_mb_to_sectors ${lvm_start} 1)"
    echo "lvm_start            $(convert_mb_to_sectors ${lvm_start} 0)            lvm_end              $(convert_mb_to_sectors ${lvm_end} 1)"
    #####


    printf 'Fix\n' | parted ---pretend-input-tty ${DEST_DISK} resizepart $emt_persistent_partition "$(convert_mb_to_sectors "${emt_persistent_end_mb}" 1)"s
    check_return_value $? "Failed to resize emt persistent"

    parted -s ${DEST_DISK} \
        mkpart hashmap_a ext4 "$(convert_mb_to_sectors "${root_hashmap_a_start}" 0)"s "$(convert_mb_to_sectors "${root_hashmap_b_start}" 1)"s \
        mkpart hashmap_b ext4 "$(convert_mb_to_sectors "${root_hashmap_b_start}" 0)"s "$(convert_mb_to_sectors "${rootfs_b_start}" 1)"s \
        mkpart rootfs_b ext4 "$(convert_mb_to_sectors "${rootfs_b_start}" 0)"s "$(convert_mb_to_sectors "${roothash_start}" 1)"s \
        mkpart roothash ext4 "$(convert_mb_to_sectors "${roothash_start}" 0)"s "$(convert_mb_to_sectors "${swap_start}" 1)"s \
        mkpart swap linux-swap "$(convert_mb_to_sectors "${swap_start}" 0)"s "$(convert_mb_to_sectors "${reserved_start}" 1)"s \
	mkpart reserved ext4  "$(convert_mb_to_sectors "${reserved_start}" 0)"s "$(convert_mb_to_sectors "$((reserved_start + reserved_size))" 1)"s

    check_return_value $? "Failed to create partitions"

    if [ $single_hdd -eq 0 ];
    then
        parted -s ${DEST_DISK} \
            mkpart lvm ext4 "$(convert_mb_to_sectors "${lvm_start}" 0)"s "$(convert_mb_to_sectors "${lvm_end}" 1)"s
        check_return_value $? "Failed to create LVM partition"
    fi

    suffix=$(fix_partition_suffix)

    mkfs -t ext4 -L roothash -F "${DEST_DISK}${suffix}${roothash_partition}"
    check_return_value $? "Failed to mkfs roothash"

    mkfs -t ext4 -L root_b -F "${DEST_DISK}${suffix}${rootfs_b_partition}"
    check_return_value $? "Failed to mkfs rootfs_b"

}

#####################################################################################
create_single_hdd_lvmg() {
    if [ $single_hdd -eq 0 ];
    then

	dd if=/dev/zero of="${DEST_DISK}${suffix}${singlehdd_lvm_partition}" bs=100MB count=1
	pvcreate -ff "${DEST_DISK}${suffix}${singlehdd_lvm_partition}"
	check_return_value $? "Failed to make mkfs ext4 on lvmvg_crypt"

	vgcreate lvmvg "${DEST_DISK}${suffix}${singlehdd_lvm_partition}"
	check_return_value $? "Failed to create a lvmvg group"
	echo "vgcreate is completed"
    fi

}

#####################################################################################
# create a logical encrypted volume with 512 sector size. This will ensure that the
# lvm that is created for openebs will be created with a 512 block size.
# This logic is needed only if there are heterogeneous block sizes.
# the output of this function is to update the global var update_sector if needed
block_disk_phy_block_disk() {
    # list_block_devices=($(lsblk -o NAME,TYPE | grep -i disk  | awk  '$1 ~ /sd*|nvme*/ {print $1}'))

    list_block_devices=($(lsblk -o NAME,TYPE,SIZE,RM | grep -i disk | awk '$1 ~ /sd*|nvme*/ {if ($3 !="0B" && $4 ==0)  {print $1}}'))
    list_of_lvmg_part=''
    block_size_4k=0
    block_size_512=0
    size_4k=0
    size_512=0

    for block_dev in ${list_block_devices[@]};
    do
	grep -i "${DEST_DISK}" <<< "/dev/${block_dev}"
	if [ $? -eq 0 ]
	then
	    continue
	fi

	# get info if there is a 4kB physical block present
	parted -s "/dev/${block_dev}" print | grep -i sector | grep -q 4098.$
	if [ $? -eq 0 ];
	then
	    block_size_4k=$(( 1 + block_size_4k ))
	    disk_4k="$disk_4k /dev/${block_dev}"
	    export disk_4k
	fi

	parted -s "/dev/${block_dev}" print | grep -i sector | grep -q 512.$
	if [ $? -eq 0 ];
	then
	    block_size_512=$(( 1 + block_size_512 ))
	    disk_512="$disk_512 /dev/${block_dev}"
	    export disk_512
	fi
	echo "512 $block_size_512"
    done

    echo "Total 4kB phy sectors block disk $block_size_4k $disk_4k"
    echo "Total 512B phy sectors block disk $block_size_512 $disk_512"

    if [ $block_size_512 -ne 0 ] && [ $block_size_4k -ne 0 ];
    then
	export UPDATE_SECTOR="--sector-size 512"
    fi
}

#####################################################################################
update_lvmvg() {
    # this is a fallback mech when lvm group create might have failed with incorrect
    # block sizes. this will not handle any other failure cases.

    size_512=0
    size_4k=0

    list_of_lvmg_part_512=''
    list_of_lvmg_part_4k=''
    #bigger lvm will be used by orchestrator
    for disk in $list_of_lvmg_part;
    do
	size=$(lsblk -b --output SIZE -n -d "${disk}")
	parted -s "${disk}" print | grep -i sector | grep -q 512.$
	if [ $? -eq 0 ];
	then
	    size_512=$(( size_512 + size ))
	    list_of_lvmg_part_512+=" ${disk} "
	else
	    size_4k=$(( size_4k + size ))
	    list_of_lvmg_part_4k+=" ${disk} "
	fi
    done

    if [ $size_4k -gt $size_512 ];
    then
	echo "Selected 4k block sized disks because of higher total size"
	if [[ $list_of_lvmg_part_4k != '' ]];
	then
	    vgcreate lvmvg $list_of_lvmg_part_4k
	    check_return_value $? "Failed to create LVMVG with 4k blocks"
	fi

	if [[ $list_of_lvmg_part_512 != '' ]];
	then
	    vgcreate lvmvg2 $list_of_lvmg_part_512
	    check_return_value $? "Failed to create LVMVG with 512 blocks"
	fi
    else
	if [[ $list_of_lvmg_part_512 != '' ]];
	then
	    vgcreate lvmvg $list_of_lvmg_part_512
	    check_return_value $? "Failed to create LVMVG-2 with 512 blocks"
	fi

	if [[ $list_of_lvmg_part_4k != '' ]];
	then
	    vgcreate lvmvg2 $list_of_lvmg_part_4k
	    check_return_value $? "Failed to create LVMVG-2 with 4k blocks"
	fi
    fi

}

#####################################################################################
partition_other_devices() {
    block_disk_phy_block_disk
    # check all disks
    # list_block_devices=($(lsblk -o NAME,TYPE | grep -i disk  | awk  '$1 ~ /sd*|nvme*/ {print $1}'))

    ## $3 represents the block device size. if 0 omit
    ## $4 is set to 1 if the device is removable
    list_block_devices=($(lsblk -o NAME,TYPE,SIZE,RM | grep -i disk | awk '$1 ~ /sd*|nvme*/ {if ($3 !="0B" && $4 ==0)  {print $1}}'))
    list_of_lvmg_part=''
    for block_dev in ${list_block_devices[@]};
    do
	grep -i "${DEST_DISK}" <<< "/dev/${block_dev}"
	if [ $? -eq 0 ]
	then
	   continue
	fi

	#Delete all partitions on that disk to make it ready for luks with 1 partition only
	line_num=$(parted -s "/dev/${block_dev}" print | awk '$1 == "Number" { print NR }')
	partition_num=$(parted -s "/dev/${block_dev}" print | awk 'NR > $line_num { print $1}')
	for part in $partition_num;
	do
	    echo "partition in $disk $part will be deleted"
	    rm_part=$(parted -s "/dev/${block_dev}" rm "$part")
	done

	# new partition
	parted -s "/dev/${block_dev}" \
	       mklabel gpt \
	       mkpart primary ext4 0% 100%

	check_return_value $? "Failed to run parted for /dev/${block_dev}"

	part_suffix=$(get_partition_suffix "/dev/${block_dev}" )

	dd if=/dev/zero of="/dev/${block_dev}${part_suffix}1" bs=100MB count=1
	pvcreate "/dev/${block_dev}${part_suffix}1"
	check_return_value $? "Failed to make mkfs ext4 on /dev/${block_dev}${part_suffix}1"

	list_of_lvmg_part+=" /dev/${block_dev}${part_suffix}1"

    done

    if [[ $list_of_lvmg_part != '' ]];
    then
	vgcreate lvmvg $list_of_lvmg_part
	if [ $? -ne 0 ]
	then
	    export list_of_lvmg_part=$list_of_lvmg_part
	    echo "Failed to create a lvmvg group"
	    echo "Trying with separated sectors"
	    update_lvmvg
	fi
	echo "vgcreate is completed"
    fi

}

#####################################################################################
format_verity_part() {

    mkfs.ext4 -F "${DEST_DISK}${suffix}${roothash_partition}"
    check_return_value $? "Failed to make mkfs ext4 on ver_roothash"

    ##############

    mkfs.ext4 -F "${DEST_DISK}${suffix}${rootfs_b_partition}"
    check_return_value $? "Failed to make mkfs ext4 on rootfs_b"
    ##############
}

#####################################################################################
enable_dmv(){
    suffix=$(fix_partition_suffix)

    #swap space creation
    mkswap  "${DEST_DISK}${suffix}${swap_partition}"
    check_return_value $? "Failed to mkswap"
    swap_uuid=$(blkid| grep swap | awk '{print $2}'|awk -F= '{print $2}' | tr -d '"')


    format_verity_part

    resize2fs -f "${DEST_DISK}${suffix}${emt_persistent_partition}"
    check_return_value $? "Failed to resize fs of persistent partition"

    # mounts needed to make the chroot work
    mount "${DEST_DISK}${suffix}${rootfs_partition}" /mnt
    check_return_value $? "Failed to mount rootfs"

    mount "${DEST_DISK}${suffix}${boot_partition}" /mnt/boot/efi
    check_return_value $? "Failed to mount /boot/efi"

    mount --bind /dev /mnt/dev
    check_return_value $? "Failed to bind /dev for chroot"

    mount --bind /dev/pts /mnt/dev/pts
    check_return_value $? "Failed to bind /dev/pts for chroot"

    mount --bind /proc /mnt/proc
    check_return_value $? "Failed to bind /proc for chroot"

    mount --bind /sys /mnt/sys
    check_return_value $? "Failed to bind /sys for chroot"

    echo "UUID=$swap_uuid swap swap default 0 2" >> /mnt/etc/fstab

    create_single_hdd_lvmg
    if ! $TEST_ON_ONLY_ONE_PART;
    then
	partition_other_devices
    fi


    chroot /mnt /bin/bash <<EOT

    #inside installed Edge Microvisor Toolkit

    # selinux relabel all the files that were touched till now by provisioning
    setfiles -m -v /etc/selinux/targeted/contexts/files/file_contexts /
    setfiles -m -v /etc/selinux/targeted/contexts/files/file_contexts /etc/hosts
    setfiles -m -v /etc/selinux/targeted/contexts/files/file_contexts  /var/lp/pua
    setfiles -m -v /etc/selinux/targeted/contexts/files/file_contexts /etc/intel_manageability.conf_bak
    setfiles -m -v /etc/selinux/targeted/contexts/files/file_contexts /etc/intel_manageability.conf
    setfiles -m -v /etc/selinux/targeted/contexts/files/file_contexts /etc/dispatcher.environment
    setfiles -m -v /etc/selinux/targeted/contexts/files/file_contexts /var/log/inbm-update-log.log
    setfiles -m -v /etc/selinux/targeted/contexts/files/file_contexts /var/log/inbm-update-status.log
    setfiles -m -v /etc/selinux/targeted/contexts/files/file_contexts /var/lib/dispatcher
    setfiles -m -v /etc/selinux/targeted/contexts/files/file_contexts /etc/intel-manageability
    setfiles -m -v /etc/selinux/targeted/contexts/files/file_contexts /var/cache/manageability
    setfiles -m -v /etc/selinux/targeted/contexts/files/file_contexts /var/intel-manageability
    setfiles -m -v /etc/selinux/targeted/contexts/files/file_contexts /var/lib/rancher
    setfiles -m -v /etc/selinux/targeted/contexts/files/file_contexts /etc/kubernetes
    setfiles -m -v /etc/selinux/targeted/contexts/files/file_contexts /etc/cni
    setfiles -m -v /etc/selinux/targeted/contexts/files/file_contexts /etc/netplan
    setfiles -m -v /etc/selinux/targeted/contexts/files/file_contexts /etc/rancher
    setfiles -m -v /etc/selinux/targeted/contexts/files/file_contexts /etc/sysconfig
    setfiles -m -v /etc/selinux/targeted/contexts/files/file_contexts /etc/cloud
    setfiles -m -v /etc/selinux/targeted/contexts/files/file_contexts /etc/udev
    setfiles -m -v /etc/selinux/targeted/contexts/files/file_contexts /etc/systemd
    setfiles -m -v /etc/selinux/targeted/contexts/files/file_contexts /etc/ssh
    setfiles -m -v /etc/selinux/targeted/contexts/files/file_contexts /etc/pki
    setfiles -m -v /etc/selinux/targeted/contexts/files/file_contexts /etc/machine-id
    setfiles -m -v /etc/selinux/targeted/contexts/files/file_contexts /etc/intel_edge_node
    setfiles -m -v /etc/selinux/targeted/contexts/files/file_contexts /etc/hosts
    setfiles -m -v /etc/selinux/targeted/contexts/files/file_contexts /etc/environment
    setfiles -m -v /etc/selinux/targeted/contexts/files/file_contexts /etc/fstab
    setfiles -m -v /etc/selinux/targeted/contexts/files/file_contexts /opt


    restorecon -R -v /
    restorecon -R -v  /var/lp/pua
    restorecon -R -v /etc/intel_manageability.conf_bak
    restorecon -R -v /etc/intel_manageability.conf
    restorecon -R -v /etc/dispatcher.environment
    restorecon -R -v /var/log/inbm-update-log.log
    restorecon -R -v /var/log/inbm-update-status.log
    restorecon -R -v /var/lib/dispatcher
    restorecon -R -v /etc/intel-manageability
    restorecon -R -v /var/cache/manageability
    restorecon -R -v /var/intel-manageability
    restorecon -R -v /var/lib/rancher
    restorecon -R -v /etc/kubernetes
    restorecon -R -v /etc/cni
    restorecon -R -v /etc/netplan
    restorecon -R -v /etc/rancher
    restorecon -R -v /etc/sysconfig
    restorecon -R -v /etc/cloud
    restorecon -R -v /etc/udev
    restorecon -R -v /etc/systemd
    restorecon -R -v /etc/ssh
    restorecon -R -v /etc/pki
    restorecon -R -v /etc/machine-id
    restorecon -R -v /etc/intel_edge_node
    restorecon -R -v /etc/hosts
    restorecon -R -v /etc/environment
    restorecon -R -v /etc/fstab
    restorecon -R -v /opt

EOT




    #cleanup of mounts
    mount_points=($(grep -i "/mnt"  /proc/mounts | awk '{print $2}' | sort -nr))
    for mounted_dir in ${mount_points[@]};
    do
	umount $mounted_dir
    done
    echo "Completed all umounts"

    #############################
    #test for B part

    if $TEST_ENABLE_DM_ON_ROOTFSB;
    then
	#backup using dd
	rootfs_dd_count=$(fdisk -l ${DEST_DISK} | grep "${DEST_DISK}${suffix}${rootfs_partition}" | awk '{print int( ($4/2048/4) + 0.999999) }')
	dd if="${DEST_DISK}${suffix}${rootfs_partition}" of="${DEST_DISK}${suffix}${rootfs_b_partition}" bs=4M count=$rootfs_dd_count status=progress
	e2fsck -fy  "${DEST_DISK}${suffix}${rootfs_partition}"
	sync
	###############
    fi
    #############################

    if $COMPLETE_DMVERITY;
    then
	mkdir /temp

	mount "${DEST_DISK}${suffix}${roothash_partition}" /temp
	check_return_value $? "Failed to mount rootfs"

        set -o pipefail
	veritysetup format "${DEST_DISK}${suffix}${rootfs_partition}" "${DEST_DISK}${suffix}${root_hashmap_a_partition}" | grep Root | cut -f2 > /temp/part_a_roothash
	check_return_value $? "Failed to do veritysetup"

	if $TEST_ENABLE_DM_ON_ROOTFSB;
	then
	    set -o pipefail
	    veritysetup format "${DEST_DISK}${suffix}${rootfs_b_partition}" "${DEST_DISK}${suffix}${root_hashmap_b_partition}" | grep Root | cut -f2 > /temp/part_b_roothash
	    check_return_value $? "Failed to do veritysetup"
	fi

	umount /temp
	rm -rf /temp
	echo "Completed veritysetup"
    fi
}


#####################################################################################
emt_main_dmv() {

    if [ -z "${DMV_IN_VEN+x}" ] || [ "$DMV_IN_VEN" = "false" ];
    then
        ven_mode_active=false
    elif [ "$DMV_IN_VEN" = "true" ];
    then 
        ven_mode_active=true
    fi

    echo "Edge Microvisor Toolkit detected"
    get_dest_disk

    is_single_hdd

    if [ "$ven_mode_active" = true ];
    then
        set_ven_partitions
	make_partition_ven
    else
	make_partition
    fi

    enable_dmv
}


emt_main_dmv
#####################################################################################
