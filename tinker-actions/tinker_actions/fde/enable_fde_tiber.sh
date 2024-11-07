#!/bin/bash
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

#####################################################################################
####
####
# COMPLETE_FDE_DMVERITY set to true if we need to encrypt all partitions
COMPLETE_FDE_DMVERITY=false

####
####
#####################################################################################
# Patition information
if $COMPLETE_FDE_DMVERITY;
then
    boot_partition=1
    rootfs_partition=2
    tiber_persistent_partition=3


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
    tiber_persistent_partition=3

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
rootfs_size=3584
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
tpm2_pcrextend='#!/bin/sh

tpm2_pcrextend '$PCR_LIST':sha256=ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff
echo "PCR '$PCR_LIST' extend completed"
'

tpm2_cryptsetup='#!/bin/sh

[ "$CRYPTTAB_TRIED" -lt "2" ] && exec tpm2-initramfs-tool unseal --pcrs '$PCR_LIST'

/usr/bin/askpass "Passphrase for $CRYPTTAB_SOURCE ($CRYPTTAB_NAME): "
'

tpm2_initramfs_tool='#!/bin/sh
PREREQ=""
prereqs()
{
   echo "$PREREQ"
}

case $1 in
prereqs)
   prereqs
   exit 0
   ;;
esac

. /usr/share/initramfs-tools/hook-functions

copy_exec /usr/lib/x86_64-linux-gnu/libtss2-tcti-device.so.0
copy_exec /usr/bin/tpm2-initramfs-tool
copy_exec /usr/bin/tpm2_pcrextend
copy_exec /usr/lib/cryptsetup/askpass /usr/bin

copy_exec /etc/initramfs-tools/tpm2-cryptsetup
'

#####################################################################################
fstab_rootfs_partition=" / ext4 discard,errors=remount-ro       0 1"
fstab_boot_partition=" /boot ext4 discard,errors=remount-ro       0 1"
fstab_efi_partition="LABEL=UEFI      /boot/efi       vfat    umask=0077      0 1"

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

    total_size_disk=$(fdisk -l ${DEST_DISK} | grep -i ${DEST_DISK} | head -1 | awk '{ print int($3)*1024}')

    # For single HDD reduce the size to 100 and fit everything inside it
    if [ $single_hdd -eq 0 ];
    then
	total_size_disk=$(( 100 * 1024 ))
    fi

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

	tiber_persistent_end=$root_hashmap_a_start
    else
	reserved_start=$(( $total_size_disk - $reserved_size ))
	tep_start=$(( $reserved_start - $tep_size ))
	swap_start=$(( $tep_start - $swap_size ))

	rootfs_b_start=$(( $swap_start - $rootfs_size ))

	tiber_persistent_end=$rootfs_b_start
    fi
    #####

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

    if $COMPLETE_FDE_DMVERITY;
    then
	parted -s ${DEST_DISK} \
	       resizepart $tiber_persistent_partition "${tiber_persistent_end}MB" \
	       mkpart hashmap_a ext4 "${root_hashmap_a_start}MB" "${root_hashmap_b_start}MB" \
	       mkpart hashmap_b ext4 "${root_hashmap_b_start}MB" "${rootfs_b_start}MB" \
	       mkpart rootfs_b ext4 "${rootfs_b_start}MB" "${roothash_start}MB" \
	       mkpart roothash ext4 "${roothash_start}MB" "${swap_start}MB" \
	       mkpart swap linux-swap "${swap_start}MB" "${tep_start}MB" \
	       mkpart tep ext4 "${tep_start}MB"  "${reserved_start}MB" \
	       mkpart reserved ext4 "${reserved_start}MB"  "${total_size_disk}MB"

	check_return_value $? "Failed to create paritions"
    else
	parted -s ${DEST_DISK} \
	       resizepart $tiber_persistent_partition "${tiber_persistent_end}MB" \
	       mkpart rootfs_b ext4 "${rootfs_b_start}MB" "${swap_start}MB" \
	       mkpart swap linux-swap "${swap_start}MB" "${tep_start}MB" \
	       mkpart tep ext4 "${tep_start}MB"  "${reserved_start}MB" \
	       mkpart reserved ext4 "${reserved_start}MB"  "${total_size_disk}MB"

	check_return_value $? "Failed to create paritions"
    fi


    if [ $single_hdd -eq 0 ];
    then
	parted -s ${DEST_DISK} \
	       mkpart lvm ext4 "${total_size_disk}MB" 100%

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
}

#####################################################################################
save_rootfs_on_ram(){
    if $COMPLETE_FDE_DMVERITY;
    then
	suffix=$(fix_partition_suffix)
	mkdir rfs
	mount "${DEST_DISK}${suffix}${rootfs_partition}" rfs
	check_return_value $? "Failed to mount rootfs"

	mkdir rfs_backup
	# mkfs -t ext4 -L reserved -F "${DEST_DISK}${suffix}${reserved_partition}"
	mount "${DEST_DISK}${suffix}${reserved_partition}" rfs_backup
	check_return_value $? "Failed to mount reserved"

	cp -rp rfs/* rfs_backup
	umount rfs
	umount rfs_backup
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
    dd if=/dev/zero of=${DEST_DISK}${suffix}${reserved_partition} bs=100MB count=20
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
	mount /dev/mapper/rootfs_crypt rfs
	check_return_value $? "Failed to mount the luks crypt for rootfs"

	mkdir -p rfs_backup
	mount "${DEST_DISK}${suffix}${reserved_partition}" rfs_backup
	check_return_value $? "Failed to mount rootfs backup"

	cp -rp rfs_backup/* rfs
	check_return_value $? "Failed to copy the rootfs back"

	rm -rf rfs_backup/*
	check_return_value $? "Failed to cleanup rfs backup"

	umount rfs
	umount rfs_backup
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

    ###luks for tiber_persistent_partition
    if $COMPLETE_FDE_DMVERITY;
    then
	mkdir -p ti_backup
	mkdir -p ti
	mount "${DEST_DISK}${suffix}${tiber_persistent_partition}" ti
	check_return_value $? "Failed to mount tiber persistent for backup"

	mount "${DEST_DISK}${suffix}${reserved_partition}" ti_backup
	check_return_value $? "Failed to mount ti_backup"

	cp -rp ti/* ti_backup
	umount ti

	luksformat_helper $luks_key "${DEST_DISK}${suffix}${tiber_persistent_partition}" "tiber_persistent"

	mkfs.ext4 -F /dev/mapper/tiber_persistent
	check_return_value $? "Failed to make mkfs ext4 on rootfs"

	mount /dev/mapper/tiber_persistent ti
	check_return_value $? "Failed to mount tiber persistent"

	cp -rp ti_backup/* ti
	check_return_value $? "Failed to copy the tiber persistent partition back"

	umount ti
	umount ti_backup
    fi
    ####

    #cleanup copied backup of rfs
    if $COMPLETE_FDE_DMVERITY;
    then
	cleanup_rfs_backup &

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
    partition_other_devices

    # Was added in ubuntu to solve some issue with resolv
    # rm /mnt/etc/resolv.conf
    # touch /mnt/etc/resolv.conf
    # mount --bind /etc/resolv.conf /mnt/etc/resolv.conf

    # mount /dev/mapper/rootfs_b_crypt rfs
    # cp -rp rfs_backup/* rfs
    # umount rfs
    # umount rfs_backup

    chroot /mnt /bin/bash <<EOT

    #inside installed ubuntu

    #setup tpm
    tpm2-initramfs-tool seal --data $(cat /luks_key) --pcrs 15
    if [ $? -ne 0 ]
    then
	echo "tpm2-initramfs-tools failed"
	exit 1
    fi

    rm -rf /luks_key

EOT




    #cleanup of mounts
    mount_points=($(grep -i "/mnt"  /proc/mounts | awk '{print $2}' | sort -nr))
    for mounted_dir in ${mount_points[@]};
    do
	umount $mounted_dir
    done


    if $COMPLETE_FDE_DMVERITY;
    then
	mkdir /temp

	mount /dev/mapper/ver_roothash /temp
	check_return_value $? "Failed to mount rootfs"

	veritysetup format /dev/mapper/rootfs_crypt /dev/mapper/root_a_ver_hash_map | grep Root | cut -f2 > /temp/part_a_roothash
	check_return_value $? "Failed to do veritysetup"

	# veritysetup format /dev/mapper/rootfs_b_crypt /dev/mapper/root_b_ver_hash_map | grep Root | cut -f2 > /temp/part_b_roothash
	# check_return_value $? "Failed to do veritysetup"

	umount /temp
	rm -rf /temp
    fi
}






#####################################################################################
tiber_main() {

    echo "Tiber OS detected"
    get_dest_disk

    is_single_hdd

    make_partition

    save_rootfs_on_ram

    enable_luks
}


tiber_main
#####################################################################################
