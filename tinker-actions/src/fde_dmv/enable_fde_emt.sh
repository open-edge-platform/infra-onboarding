#!/bin/bash

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

#####################################################################################
####
####
# COMPLETE_FDE_DMVERITY set to true if we need to encrypt all partitions
COMPLETE_FDE_DMVERITY=true

# Test flag for enabling DM-verity on B part aswell.
TEST_ENABLE_DM_ON_ROOTFSB=false

# Test flag for only partition
TEST_ON_ONLY_ONE_PART=false

# Get the user provided lvm disk size number
MINIMUM_LVM_SIZE=0

####
####
#####################################################################################
# Patition information
if $COMPLETE_FDE_DMVERITY;
then
    boot_partition=1
    rootfs_partition=2
    emt_persistent_partition=3

    #efi_partition=15

    root_hashmap_a_partition=4
    root_hashmap_b_partition=5
    rootfs_b_partition=6
    roothash_partition=7

    swap_partition=8
    tep_partition=9
    reserved_partition=10
    singlehdd_lvm_partition=11
else
    boot_partition=1
    rootfs_partition=2
    emt_persistent_partition=3

    rootfs_b_partition=4

    swap_partition=5
    tep_partition=6
    reserved_partition=7
    singlehdd_lvm_partition=8
fi

# DEST_DISK set from the template.yaml file as an environment variable.

#####################################################################################
# Partitions in %
swap_space_start=91

# Size in MBs
tep_size=14336
reserved_size=5120
boot_size=5120600
bare_min_rootfs_size=25
rootfs_size=4096
rootfs_hashmap_size=100
rootfs_roothash_size=50


#####################################################################################
#Global var which is updated
single_hdd=-1
check_all_disks=1
#####################################################################################
luks_key=$PWD/luks_key

#PCR_LIST example - PCR_LIST=0,2,7
PCR_LIST=15

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

    #if there were any problems when the ubuntu was streamed.
    # printf 'Fix\n' | parted ---pretend-input-tty -l

    #ram_size
    ram_size=$(free -g | grep -i mem | awk '{ print $2 }' )
    swap_size=$(( $ram_size / 2 ))


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

    swap_size=$(( $swap_size * 1024 ))

    total_size_disk=$(lsblk -b -dn -o SIZE "$DEST_DISK" | awk '{ printf "%.0f\n", $1 / (1024*1024) }')

    # For single HDD Size should be total disk - lvm_size in GB provided as input by the User
    if [ $single_hdd -eq 0 ];
    then
        if [ $lvm_disk_size -ge 1 ];
        then
            min_lvm_size=$(( lvm_disk_size*1024 ))
            if [ "$min_lvm_size" -ge "$total_size_disk" ];
            then
                check_return_value 1 "$lvm_size is more than the disk size,please check"
            fi
        fi
	total_size_disk=$(( 100 * 1024 ))
    fi
    echo "total_size_disk(fixed) ${total_size_disk}"

    #####
    if $COMPLETE_FDE_DMVERITY;
    then
	reserved_start=$(( $total_size_disk - $reserved_size ))
	tep_start=$(( $reserved_start - $tep_size ))
	swap_start=$(( $tep_start - $swap_size ))

	roothash_start=$(( $swap_start - $rootfs_roothash_size ))
	rootfs_b_start=$(( $roothash_start - $rootfs_size ))
	root_hashmap_b_start=$(( $rootfs_b_start - $rootfs_hashmap_size ))
	root_hashmap_a_start=$(( $root_hashmap_b_start - $rootfs_hashmap_size ))

	emt_persistent_end=$root_hashmap_a_start
    else
	reserved_start=$(( $total_size_disk - $reserved_size ))
	tep_start=$(( $reserved_start - $tep_size ))
	swap_start=$(( $tep_start - $swap_size ))

	rootfs_b_start=$(( $swap_start - $rootfs_size ))

	emt_persistent_end=$rootfs_b_start
    fi
    #####

    #save this size of emt persistent before partition
    suffix=$(fix_partition_suffix)
    export emt_persistent_dd_count=$(fdisk -l ${DEST_DISK} | grep "${DEST_DISK}${suffix}${emt_persistent_partition}" | awk '{print int( ($4/2048/4) + 0.999999) }')
    #####
    # logging needed to understand the block splits
    echo "DEST_DISK ${DEST_DISK}"
    echo "rootfs_partition     $rootfs_partition         rootfs_end           ${rootfs_end}MB"
    echo "root_hashmap_a_start ${root_hashmap_a_start}MB root_hashmap_b_start ${root_hashmap_b_start}MB"
    echo "root_hashmap_b_start ${root_hashmap_b_start}MB rootfs_b_start       ${rootfs_b_start}MB"
    echo "rootfs_b_start       ${rootfs_b_start}MB       roothash_start       ${roothash_start}MB"
    echo "roothash_start       ${roothash_start}MB       swap_start           ${swap_start}MB"
    echo "swap_start          ${swap_start}MB            tep_start            ${tep_start}MB"
    echo "tep_start           ${tep_start}MB             reserved_start       ${reserved_start}MB"
    echo "reserved_start      ${reserved_start}MB        total_size_disk      ${total_size_disk}MB"
    #####

    echo "sizes in sectors"
    echo "rootfs_partition     $rootfs_partition         rootfs_end             $(convert_mb_to_sectors ${rootfs_end} 1)"
    echo "root_hashmap_a_start $(convert_mb_to_sectors ${root_hashmap_a_start} 0) root_hashmap_b_start $(convert_mb_to_sectors ${root_hashmap_b_start} 1)"
    echo "root_hashmap_b_start $(convert_mb_to_sectors ${root_hashmap_b_start} 0) rootfs_b_start       $(convert_mb_to_sectors ${rootfs_b_start} 1)"
    echo "rootfs_b_start       $(convert_mb_to_sectors ${rootfs_b_start} 0)       roothash_start       $(convert_mb_to_sectors ${roothash_start} 1)"
    echo "roothash_start       $(convert_mb_to_sectors ${roothash_start} 0)       swap_start           $(convert_mb_to_sectors ${swap_start} 1)"
    echo "swap_start          $(convert_mb_to_sectors ${swap_start} 0)            tep_start            $(convert_mb_to_sectors ${tep_start} 1)"
    echo "tep_start           $(convert_mb_to_sectors ${tep_start} 0)             reserved_start       $(convert_mb_to_sectors ${reserved_start} 1)"
    echo "reserved_start      $(convert_mb_to_sectors ${reserved_start} 0)        total_size_disk      $(convert_mb_to_sectors ${total_size_disk} 1)"
    #####

    if $COMPLETE_FDE_DMVERITY;
    then
	#this cmd only resizes parition. if there is an error this should handle it.
	printf 'Fix\n' | parted ---pretend-input-tty ${DEST_DISK} \
	       resizepart $emt_persistent_partition $(convert_mb_to_sectors "${emt_persistent_end}" 1)s

	check_return_value $? "Failed to resize emt persistent paritions"

	#this cmd only creates new partitions.
	parted -s ${DEST_DISK} \
	       mkpart hashmap_a ext4  $(convert_mb_to_sectors "${root_hashmap_a_start}" 0)s $(convert_mb_to_sectors "${root_hashmap_b_start}" 1)s \
	       mkpart hashmap_b ext4  $(convert_mb_to_sectors "${root_hashmap_b_start}" 0)s $(convert_mb_to_sectors "${rootfs_b_start}" 1)s \
	       mkpart rootfs_b ext4   $(convert_mb_to_sectors "${rootfs_b_start}" 0)s       $(convert_mb_to_sectors "${roothash_start}" 1)s \
	       mkpart roothash ext4   $(convert_mb_to_sectors "${roothash_start}" 0)s       $(convert_mb_to_sectors "${swap_start}" 1)s \
	       mkpart swap linux-swap $(convert_mb_to_sectors "${swap_start}" 0)s           $(convert_mb_to_sectors "${tep_start}" 1)s \
	       mkpart tep ext4        $(convert_mb_to_sectors "${tep_start}" 0)s            $(convert_mb_to_sectors "${reserved_start}" 1)s \
	       mkpart reserved ext4   $(convert_mb_to_sectors "${reserved_start}" 0)s       $(convert_mb_to_sectors "${total_size_disk}" 1)s


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

    # Create LVM for single_hdd only when user chooses
    if [ $single_hdd -eq 0 ];
    then
	if [ $lvm_disk_size -ge 1 ];
        then
	    actual_disk_size=$(lsblk -b -dn -o SIZE "$DEST_DISK" | awk '{ printf "%.0f\n", $1 / (1024*1024) }')
	    disk_used_mb=$(lsblk -b -n -o NAME,SIZE "$DEST_DISK" \
                    | awk -v disk="$(basename "$DEST_DISK")" '$1 != disk {s+=$2} END {printf "%.0f", s / 1024 / 1024}')
	    available_disk_space=$(( actual_disk_size - disk_used_mb ))

	    if [ "$min_lvm_size" -ge "$available_disk_space" ];
            then
		echo "Available LVM size is  ${available_lvm_space}MB only."
	    else
		echo "Minimum LVM size is available."
	    fi
	fi
	parted -s ${DEST_DISK} \
		mkpart lvm ext4 "$(convert_mb_to_sectors "${total_size_disk}" 0)"s 100%
	check_return_value $? "Failed to create lvm parition"
    fi

    suffix=$(fix_partition_suffix)

    #TEP partition is not formated currently.
    #reserved
    mkfs -t ext4 -L reserved -F "${DEST_DISK}${suffix}${reserved_partition}"
    check_return_value $? "Failed to mkfs reserved"

    if $COMPLETE_FDE_DMVERITY;
    then
	#roothash partition
	mkfs -t ext4 -L roothash -F "${DEST_DISK}${suffix}${roothash_partition}"
	check_return_value $? "Failed to mkfs roothash part"

	# rootfs for a/B updated
	mkfs -t ext4 -L root_b -F "${DEST_DISK}${suffix}${rootfs_b_partition}"
	check_return_value $? "Failed to mkfs rootfs part"
    fi

    # Save the emt persistent
    # this is needed because we need to resize the rootfs a
    ##############
    #save using dd
    dd if="${DEST_DISK}${suffix}${emt_persistent_partition}" of="${DEST_DISK}${suffix}${reserved_partition}" bs=4M status=progress conv=sparse count=$emt_persistent_dd_count
    sync
    ##############

    # delete the complete emt persistent partition
    parted -s ${DEST_DISK} \
	   rm "${emt_persistent_partition}"

    #resize rootfs a partition
    rootfs_a_start=$(parted ${DEST_DISK} unit MB print | awk '/^ '$rootfs_partition'/ {gsub(/MB/, "", $2); print $2}')

    #end size of rootfs a partition
    rootfs_a_end=$(( rootfs_a_start + rootfs_size ))

    # resize part a
    parted -s ${DEST_DISK} \
	   resizepart $rootfs_partition "$(convert_mb_to_sectors "${rootfs_a_end}" 1)"s \
	   mkpart edge_persistent ext4 "$(convert_mb_to_sectors "${rootfs_a_end}" 0)"s "$(convert_mb_to_sectors "${emt_persistent_end}" 1)"s

    # restore the copied data from reserved
    #backup using dd
    dd if="${DEST_DISK}${suffix}${reserved_partition}" of="${DEST_DISK}${suffix}${emt_persistent_partition}" bs=4M status=progress conv=sparse count=$emt_persistent_dd_count
    sync
    ##############
}

#####################################################################################
save_rootfs_on_ram(){
    if $COMPLETE_FDE_DMVERITY;
    then
	suffix=$(fix_partition_suffix)
	export rootfs_dd_count=$(fdisk -l ${DEST_DISK} | grep "${DEST_DISK}${suffix}${rootfs_partition}" | awk '{print int( ($4/2048/4) + 0.999999) }')

	#############
	#save using dd
	dd if="${DEST_DISK}${suffix}${rootfs_partition}" of="${DEST_DISK}${suffix}${reserved_partition}" bs=4M count=$rootfs_dd_count status=progress
	sync
	#############
    fi
}

#####################################################################################
create_single_hdd_lvmg() {
    if [ $single_hdd -eq 0 ];
    then
	luksformat_helper $luks_key "${DEST_DISK}${suffix}${singlehdd_lvm_partition}" "lvmvg_crypt"

	pvcreate "/dev/mapper/lvmvg_crypt"
	check_return_value $? "Failed to make mkfs ext4 on lvmvg_crypt"

	vgcreate lvmvg "/dev/mapper/lvmvg_crypt"
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
	    block_size_4k=$(( 1 + $block_size_4k ))
	    export disk_4k="$disk_4k /dev/${block_dev}"
	fi

	parted -s "/dev/${block_dev}" print | grep -i sector | grep -q 512.$
	if [ $? -eq 0 ];
	then
	    block_size_512=$(( 1 + $block_size_512 ))
	    export disk_512="$disk_512 /dev/${block_dev}"
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
	    size_512=$(( $size_512 + $size ))
	    list_of_lvmg_part_512+=" ${disk} "
	else
	    size_4k=$(( $size_4k + $size ))
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

	cryptsetup luksFormat  \
		   --batch-mode \
		   --pbkdf-memory=2097152 \
		   --pbkdf-parallel=8  \
		   --cipher=aes-xts-plain64 \
		   --reduce-device-size 32M $UPDATE_SECTOR\
		   "/dev/${block_dev}${part_suffix}1" \
		   $luks_key

	check_return_value $? "Failed to luks format for /dev/${block_dev}${part_suffix}1"

	cryptsetup luksOpen "/dev/${block_dev}${part_suffix}1" "${block_dev}_crypt" --key-file=$luks_key
	check_return_value $? "Failed to luks open ${block_dev}${part_suffix}1_crypt"

	pvcreate "/dev/mapper/${block_dev}_crypt"
	check_return_value $? "Failed to make mkfs ext4 on ${block_dev}_crypt"

	list_of_lvmg_part+=" /dev/mapper/${block_dev}_crypt"

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
cleanup_rfs_backup() {
    # running this as part of another process to speed up the FDE
    dd if=/dev/zero of=${DEST_DISK}${suffix}${reserved_partition} bs=100MB count=1
}

#####################################################################################
luks_format_verity_part() {
    lukskey=$1

    ##############
    luksformat_helper $luks_key "${DEST_DISK}${suffix}${root_hashmap_a_partition}" "root_a_ver_hash_map"

    ##############
    luksformat_helper $luks_key "${DEST_DISK}${suffix}${root_hashmap_b_partition}" "root_b_ver_hash_map"

    ##############
    luksformat_helper $luks_key "${DEST_DISK}${suffix}${roothash_partition}" "ver_roothash"

    mkfs.ext4 -F /dev/mapper/ver_roothash
    check_return_value $? "Failed to make mkfs ext4 on ver_roothash"

    ##############
    #rootfs b part
    luksformat_helper $luks_key "${DEST_DISK}${suffix}${rootfs_b_partition}" "rootfs_b_crypt"

    mkfs.ext4 -F /dev/mapper/rootfs_b_crypt
    check_return_value $? "Failed to make mkfs ext4 on rootfs_b_crypt"
    ##############
}

#####################################################################################
update_luks_uuid() {

    suffix=$(fix_partition_suffix)

    mkdir -p /boot_efi_mount
    mount "$DEST_DISK${suffix}${boot_partition}" /boot_efi_mount
    rootfs_a_UUID=$( grep -a -h -o "boot_uuid=.* " /boot_efi_mount/EFI/Linux/* | cut -c 11-46 | head -1)
    umount /boot_efi_mount

    echo "YES" | cryptsetup luksUUID "$DEST_DISK${suffix}${rootfs_partition}" --uuid $rootfs_a_UUID
    check_return_value $? "Failed to set the UUID for the rootfs a partition(part2)"

}

#####################################################################################
luksformat_helper(){
    luks_key=$1
    partition=$2
    dm_name=$3
    cryptsetup luksFormat  \
	       --batch-mode \
	       --pbkdf-memory=2097152 \
	       --pbkdf-parallel=8  \
	       --cipher=aes-xts-plain64 \
	       --reduce-device-size 32M \
	       $partition \
	       $luks_key
    check_return_value $? "Failed to luks format $partition"

    cryptsetup luksOpen $partition $dm_name --key-file=$luks_key
    check_return_value $? "Failed to luks open $partition $dm_name"
}
#####################################################################################
enable_luks(){
    suffix=$(fix_partition_suffix)

    tpm2_getrandom 32 | xxd -p -c999 | tr -d '\n' > $luks_key
    check_return_value $? "Failed to get random number"

    # luks format rootfs
    if $COMPLETE_FDE_DMVERITY;
    then
	luksformat_helper $luks_key "${DEST_DISK}${suffix}${rootfs_partition}" "rootfs_crypt"

	mkfs.ext4 -F /dev/mapper/rootfs_crypt
	check_return_value $? "Failed to make mkfs ext4 on rootfs"

	mkdir -p rfs

	mkdir -p rfs_backup

	###############

	# Get the total number of blocks
	total_blocks=$(dumpe2fs -h /dev/mapper/rootfs_crypt | grep 'Block count' | awk '{print $3}')

	# Resize the filesystem
	e2fsck -fy  "${DEST_DISK}${suffix}${reserved_partition}"
	check_return_value $? "Failed to check fs on reserved partition"

	resize2fs -f "${DEST_DISK}${suffix}${reserved_partition}" $total_blocks
	check_return_value $? "Failed to resize2fs reserved for rootfs"

	#backup using dd
	dd if="${DEST_DISK}${suffix}${reserved_partition}" of=/dev/mapper/rootfs_crypt bs=4M count=$rootfs_dd_count status=progress
	sync
	###############

    fi

    ### setup swap luks
    luksformat_helper $luks_key "${DEST_DISK}${suffix}${swap_partition}" "swap_crypt"

    #swap space creation
    mkswap /dev/mapper/swap_crypt
    check_return_value $? "Failed to mkswap"

    ### setup swap luks completed

    ###luks for rootfs hash
    if $COMPLETE_FDE_DMVERITY;
    then
	luks_format_verity_part $luks_key
    fi

    ###luks for emt_persistent_partition
    if $COMPLETE_FDE_DMVERITY;
    then

	echo "$emt_persistent_dd_count emt_persistent_dd_count"
	##############	
	#save using dd
	dd if="${DEST_DISK}${suffix}${emt_persistent_partition}" of="${DEST_DISK}${suffix}${reserved_partition}" bs=4M status=progress conv=sparse count=$emt_persistent_dd_count
	sync
	##############

	luksformat_helper $luks_key "${DEST_DISK}${suffix}${emt_persistent_partition}" "emt_persistent"

	mkfs.ext4 -F /dev/mapper/emt_persistent
	check_return_value $? "Failed to make mkfs ext4 on rootfs"

	###############
	# Get the total number of blocks
	total_blocks=$(dumpe2fs -h /dev/mapper/emt_persistent | grep 'Block count' | awk '{print $3}')

	#backup using dd
	dd if="${DEST_DISK}${suffix}${reserved_partition}" of=/dev/mapper/emt_persistent bs=4M status=progress conv=sparse count=$emt_persistent_dd_count
	sync
	###############

	# Resize the filesystem on emt persistent because we cant increase a size beyond the phy blocks
	e2fsck -fy  /dev/mapper/emt_persistent
	check_return_value $? "Failed to check fs on reserved for emt persistent"

	resize2fs -f /dev/mapper/emt_persistent $total_blocks
	check_return_value $? "Failed to resize2fs reserved for rootfs"

    fi
    ####

    #cleanup copied backup of rfs
    if $COMPLETE_FDE_DMVERITY;
    then
	###############################################
	cleanup_rfs_backup &
	###############################################

	# mounts needed to make the chroot work
	mount /dev/mapper/rootfs_crypt /mnt
	check_return_value $? "Failed to mount rootfs"
    else
	# mounts needed to make the chroot work
	mount "${DEST_DISK}${suffix}${rootfs_partition}" /mnt
	check_return_value $? "Failed to mount rootfs"
    fi

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

    #temp copy needed to make the installed ubuntu to seal on the tpm
    cp $luks_key /mnt/luks_key

    create_single_hdd_lvmg
    if ! $TEST_ON_ONLY_ONE_PART;
    then
	partition_other_devices
    fi

    # updated the rootfs part uuid
    update_luks_uuid

    chroot /mnt /bin/bash <<EOT

    #inside installed Edge Microvisor Toolkit

    export TPM2TOOLS_TCTI="device:/dev/tpmrm0"

    #setup tpm
    tpm2-initramfs-tool seal --data $(cat /luks_key) --pcrs 15
    if [ $? -ne 0 ]
    then
	echo "tpm2-initramfs-tools failed"
	exit 1
    fi

    rm -rf /luks_key

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
	dd if=/dev/mapper/rootfs_crypt of=/dev/mapper/rootfs_b_crypt bs=4M count=$rootfs_dd_count status=progress
	sync
	###############
    fi
    #############################

    if $COMPLETE_FDE_DMVERITY;
    then
	mkdir /temp

	mount /dev/mapper/ver_roothash /temp
	check_return_value $? "Failed to mount rootfs"

	veritysetup format /dev/mapper/rootfs_crypt /dev/mapper/root_a_ver_hash_map | grep Root | cut -f2 > /temp/part_a_roothash
	check_return_value $? "Failed to do veritysetup"

	if $TEST_ENABLE_DM_ON_ROOTFSB;
	then
	    veritysetup format /dev/mapper/rootfs_b_crypt /dev/mapper/root_b_ver_hash_map | grep Root | cut -f2 > /temp/part_b_roothash
	    check_return_value $? "Failed to do veritysetup"
	fi

	umount /temp
	rm -rf /temp
	echo "Completed veritysetup"
    fi
}






#####################################################################################
emt_main() {

    echo "Edege Microvisor toolkit detected"
    get_dest_disk

    is_single_hdd

    if [ "$MINIMUM_LVM_SIZE" != 0 ];
    then
        lvm_disk_size=$MINIMUM_LVM_SIZE
    else
        lvm_disk_size=0
    fi

    make_partition

    save_rootfs_on_ram

    enable_luks
}


emt_main
#####################################################################################
