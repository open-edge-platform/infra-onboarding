#!/bin/bash

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

#####################################################################################
# Partition information
# partitions are updated as part of the script based on the version of Ubuntu.
# This here is a reference only.
# rootfs_partition=1
# boot_partition=2
# swap_partition=3
# tep_partition=4
# reserved_partition=5
# efi_partition=15
# singlehdd_lvm_partition=6

# DEST_DISK set from the template.yaml file as an environment variable.

# Pass LVM_SIZE to set the minimum size of LVM in GBs. The script will
# create an LVM partition of at least this size.

# Pass MINIMUM_ROOTFS_SIZE to set the minimum size of rootfs in GBs. The script will
# create an rootfs partition of at least this size.

#####################################################################################
# Partitions in %
# read as 90% or 91%
rootfs_space_end=90
boot_space_start=90
boot_space_end=91
swap_space_start=91

# Size in GBs
tep_size=14
reserved_size=5
boot_size=5
bare_min_rootfs_size=25
rootfs_size=50
persistent_size=20

#####################################################################################
#Global var which is updated
single_hdd=-1
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

get_tpm2_initramfs_tool() {
    echo "#!/bin/sh
PREREQ=\"\"
prereqs()
{
   echo \"\$PREREQ\"
}

case \$1 in
prereqs)
   prereqs
   exit 0
   ;;
esac

. /usr/share/initramfs-tools/hook-functions

copy_exec /usr/lib/x86_64-linux-gnu/libtss2-tcti-device.so.0
copy_exec /usr/bin/tpm2-initramfs-tool
copy_exec /usr/bin/tpm2_pcrextend

copy_exec ${initramfs_loc}/tpm2-cryptsetup
"
}

#####################################################################################
fstab_rootfs_partition=" / ext4 discard,errors=remount-ro       0 1"
#fstab_boot_partition=" /boot ext4 rw,relatime 0 0"
fstab_boot_partition=" /boot ext4 discard,errors=remount-ro       0 1"
# fstab_swap_partition="${DEST_DISK}${suffix}${swap_partition} none swap sw 0 0"
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
set_correct_partition() {
    export rootfs_partition=1

    mkdir -p rfs
    suffix=$(fix_partition_suffix)
    mount "${DEST_DISK}${suffix}${rootfs_partition}" rfs
    check_return_value $? "Failed to mount rootfs"

    cat rfs/etc/lsb-release | grep -i "22.04"
    if [ $? -eq 0 ];
    then
        #ubuntu 22.04 partitions
        export boot_partition=2
        export swap_partition=3
        export tep_partition=4
        export reserved_partition=5
        export persistent_partition=6
        export singlehdd_lvm_partition=7
        export efi_partition=15
        export ubuntu_version=22.04
        export initramfs_loc="/etc/initramfs-tools"
    else
        #ubuntu 24.04 partitions
        export swap_partition=2
        export tep_partition=3
        export reserved_partition=4
        export persistent_partition=5
        export singlehdd_lvm_partition=6
        export efi_partition=15
        export boot_partition=16
        export ubuntu_version=24.04
        export initramfs_loc="/usr/share/initramfs-tools"
    fi

    umount rfs
    echo "Partitions set as:"
    echo "  rootfs:        $rootfs_partition"
    echo "  boot:          $boot_partition"
    echo "  swap:          $swap_partition"
    echo "  tep:           $tep_partition"
    echo "  reserved:      $reserved_partition"
    echo "  efi:           $efi_partition"
    echo "  singlehdd_lvm: $singlehdd_lvm_partition"
}

#####################################################################################
mininum_lvm_requested() {

    echo "LVM_SIZE $LVM_SIZE"
    if [ -z "${LVM_SIZE+x}" ] || [ "$LVM_SIZE" -lt 0 ];
    then
        #default minimum lvm size is 20GB
        export lvm_size=20
        echo "LVM_SIZE set to 20GB"
    else
        if ! [[ "$LVM_SIZE" =~ ^[0-9]+$ ]]; then
            echo "LVM_SIZE must be a positive integer."
            exit 1
        fi
        echo "LVM_SIZE is set to $LVM_SIZE"
        export lvm_size=$LVM_SIZE
    fi
}

#####################################################################################
minimum_rootfs_requested() {

    echo "MINIMUM_ROOTFS_SIZE $MINIMUM_ROOTFS_SIZE"
    if [ -z "${MINIMUM_ROOTFS_SIZE+x}" ] || [ "$MINIMUM_ROOTFS_SIZE" -lt 0 ];
    then
        #default minimum rootfs size is 50GB
        export rootfs_size=50
        echo "MINIMUM_ROOTFS_SIZE set to 50GB"
    else
        if ! [[ "$MINIMUM_ROOTFS_SIZE" =~ ^[0-9]+$ ]]; then
            echo "MINIMUM_ROOTFS_SIZE must be a positive integer."
            exit 1
        fi
        echo "MINIMUM_ROOTFS_SIZE is set to $MINIMUM_ROOTFS_SIZE"
        export rootfs_size=$MINIMUM_ROOTFS_SIZE
    fi
}

#####################################################################################
partition_with_percent() {

    #ram_size
    ram_size=$(free -g | grep -i mem | awk '{ print $2 }' )
    swap_size=$(( $ram_size + 2 ))

    total_size_disk=$(parted -s ${DEST_DISK} p | grep -i ${DEST_DISK} | awk '{ print $3 }' | sed  's/GB//g' )

    swap_space_end=$(echo $swap_size $total_size_disk $swap_space_start | awk '{print int(($1*100)/$2 + $3)}' )

    #tep space is added after swap space
    tep_space_end=$(( $swap_space_end + $tep_size ))

    echo "Dest_disk=${DEST_DISK} tep_space_end=$tep_space_end swap_space_end=$swap_space_end"
    
    parted -s ${DEST_DISK} \
	   resizepart $rootfs_partition $rootfs_space_end% \
	   mkpart primary ext4 $boot_space_start% $boot_space_end% \
	   mkpart primary linux-swap $swap_space_start% $swap_space_end% \
	   mkpart primary ext4 $swap_space_end%  $tep_space_end% \
	   mkpart primary ext4 $tep_space_end%  100%

    check_return_value $? "Failed to create paritions"
    
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
make_partition_single_hdd() {

    #if there were any problems when the ubuntu was streamed.
    # printf 'Fix\n' | parted ---pretend-input-tty -l

    #ram_size
    ram_size=$(free -g | grep -i mem | awk '{ print $2 }' )

    #
    # limit swap size to sqrt of ramsize link https://help.ubuntu.com/community/SwapFaq
    #
    # this is to reconcile the requirement where we have a upper limit of 100GB for
    # all partitions apart from lvm we cant risk exceeding the swap size.
    swap_size=$(echo "$ram_size" | awk '{printf ("%.0f\n", sqrt($1))}')

    if [[ $swap_size -gt 128 ]];
    then
        swap_size=128
    fi

    total_size_disk=$(parted -s ${DEST_DISK} p | grep -i ${DEST_DISK} | awk '{ print $3 }' | sed  's/GB//g' )
    actual_disk_size=$total_size_disk

    # Size of OS with all other partitions are capped to 100GB
    #
    if [[ $total_size_disk -gt 100 ]];
    then
        #total_size_disk=100

        #Bare minimum needed to encrypt rootfs is 3GB
        reserved_size=3

        # for u24 boot size is not used would mean that rootfs_size will increase by 5GB
        total_size_disk=$(( $reserved_size + $tep_size + $swap_size + $boot_size + $rootfs_size ))

        if [[ $total_size_disk -gt $actual_disk_size ]];
        then
            total_size_disk=$actual_disk_size
            echo "Sizes are adjusted to actual disk size $actual_disk_size"
        fi

        reserved_end=$total_size_disk

    else
        minimum_size=$(( $reserved_size + $tep_size + $swap_size + $boot_size + $bare_min_rootfs_size ))
        if [[ $minimum_size -gt $total_size_disk ]];
        then
            # This entire if block is to start optimization of the each of the blocks.
            difference=$(( $minimum_size - $total_size_disk ))

            # first check for the reserved size a mininum of 2 is needed.
            if [[ $difference -gt 0 ]];
            then
            difference=$(( $difference - 3 ))
            reserved_size=$(( $reserved_size - 3 ))
            fi

            # trusted compute will be given only 2 GB in such a constrainted environment.
            if [[ $difference -gt 0 ]];
            then
            difference=$(( $difference - 12 ))
            tep_size=$(( $tep_size - 12 ))
            fi

            # last for the critical one. if the swap space is cut to less than half FDE will not proceed.
            if [[ $difference -gt 0 ]];
            then
            temp_swap_size=$(( $swap_size - $difference ))
            if [[ $temp_swap_size -lt $swap_size ]];
            then
                echo "PLATFORM CANNOT SUPPORT SWAP SPACE."
                exit 1
            fi
            fi
        fi
    fi

    #####
    lvm_start=$(( $actual_disk_size - $lvm_size))
    persistent_start=$(( $total_size_disk))

    reserved_start=$(( $total_size_disk - $reserved_size ))
    tep_start=$(( $reserved_start - $tep_size ))
    swap_start=$(( $tep_start - $swap_size ))
    if [ "$ubuntu_version" == "22.04" ];
    then
        boot_start=$(( $swap_start - $boot_size ))
        rootfs_end=$boot_start
    else 
        rootfs_end=$(( $swap_start ))
        boot_start="Not used"
    fi

    #####

    #####
    # logging needed to understand the block splits
    echo "DEST_DISK  ${DEST_DISK}"
    echo "rootfs_partition  $rootfs_partition       rootfs_end         ${rootfs_end}GB"
    echo "boot_start         ${boot_start}GB        swap_start         ${swap_start}GB"
    echo "swap_start         ${swap_start}GB        tep_start          ${tep_start}GB"
    echo "tep_start          ${tep_start}GB         reserved_start     ${reserved_start}GB"
    echo "reserved_start     ${reserved_start}GB    reserved_end       ${reserved_end}GB"
    echo "Persistent start   ${persistent_start}GB  Persistent end     ${lvm_start}GB"
    echo "LVM start          ${lvm_start}GB         actual_disk_size   ${actual_disk_size}GB"
    #####

    if [[ $lvm_size -gt 0 ]]
    then
        # to be used in the parted command only
        parted_for_lvm="mkpart lvm ext4 ${lvm_start}GB 100%"
    fi


    suffix=$(fix_partition_suffix)

    if [ "$ubuntu_version" == "22.04" ];
    then

        parted -s ${DEST_DISK} \
            resizepart $rootfs_partition "${rootfs_end}GB" \
            mkpart boot            ext4       "${boot_start}GB"     "${swap_start}GB" \
            mkpart swap            linux-swap "${swap_start}GB"     "${tep_start}GB" \
            mkpart trusted_compute ext4       "${tep_start}GB"      "${reserved_start}GB" \
            mkpart reserved        ext4       "${reserved_start}GB" "${reserved_end}GB" \
            mkpart persistent      ext4       "${persistent_start}GB" "${lvm_start}GB" \
            ${parted_for_lvm}

        check_return_value $? "Failed to create paritions"

        #/boot is now kept in a different partition
        mkfs -t ext4 -L boot -F "${DEST_DISK}${suffix}${boot_partition}"
        check_return_value $? "Failed to mkfs boot"
    else
        #ubuntu 24.04
        parted -s ${DEST_DISK} \
            resizepart $rootfs_partition "${rootfs_end}GB" \
            mkpart swap            linux-swap "${swap_start}GB"      "${tep_start}GB" \
            mkpart trusted_compute ext4       "${tep_start}GB"       "${reserved_start}GB" \
            mkpart reserved        ext4       "${reserved_start}GB"   "${reserved_end}GB" \
            mkpart persistent      ext4       "${persistent_start}GB" "${lvm_start}GB" \
            ${parted_for_lvm}

        check_return_value $? "Failed to create paritions"
    fi

    #swap space creation
    # mkswap "${DEST_DISK}${suffix}${swap_partition}"
    # check_return_value $? "Failed to mkswap"

    #TEP and reserved are not formated currently.
    #reserved
    mkfs -t ext4 -L reserved -F "${DEST_DISK}${suffix}${reserved_partition}"
    check_return_value $? "Failed to mkfs boot"
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

    total_size_disk=$(parted -s ${DEST_DISK} p | grep -i ${DEST_DISK} | awk '{ print $3 }' | sed  's/GB//g' )

    swap_space_end=$(echo $swap_size $total_size_disk $swap_space_start | awk '{print int(($1*100)/$2 + $3)}' )

    #tep space is added after swap space
    tep_space_end=$(( $swap_space_end + $tep_size ))

    echo "Dest_disk=${DEST_DISK} tep_space_end=$tep_space_end swap_space_end=$swap_space_end"

    #####
    total_required_size_disk=$(( $reserved_size + $tep_size + $swap_size + $boot_size + $rootfs_size ))
    persistent_start=$(( $total_required_size_disk))

    reserved_start=$(( $persistent_start - $reserved_size ))
    tep_start=$(( $reserved_start - $tep_size ))
    swap_start=$(( $tep_start - $swap_size ))
    boot_start=$(( $swap_start - $boot_size ))
    rootfs_end=$boot_start
    #####

    #####
    # logging needed to understand the block splits
    echo "DEST_DISK ${DEST_DISK}"
    echo "rootfs_partition  $rootfs_partition       rootfs_end       ${rootfs_end}GB"
    echo "boot_start         ${boot_start}GB        swap_start       ${swap_start}GB"
    echo "swap_start         ${swap_start}GB        tep_start        ${tep_start}GB"
    echo "tep_start          ${tep_start}GB         reserved_start   ${reserved_start}GB"
    echo "reserved_start     ${reserved_start}GB   reserved_end      ${persistent_start}GB"
    echo "Persistent start   ${persistent_start}GB  Persistent end   ${total_size_disk}GB"
    #####



    suffix=$(fix_partition_suffix)

    if [ "$ubuntu_version" == "22.04" ];
    then
        parted -s ${DEST_DISK} \
            resizepart $rootfs_partition "${rootfs_end}GB" \
            mkpart boot            ext4       "${boot_start}GB"     "${swap_start}GB" \
            mkpart swap            linux-swap "${swap_start}GB"     "${tep_start}GB" \
            mkpart trusted_compute ext4       "${tep_start}GB"      "${reserved_start}GB" \
            mkpart reserved        ext4       "${reserved_start}GB" "${persistent_start}GB" \
            mkpart persistent      ext4       "${persistent_start}GB" 100%

        check_return_value $? "Failed to create paritions"

        #/boot is now kept in a different partition
        mkfs -t ext4 -L boot -F "${DEST_DISK}${suffix}${boot_partition}"
        check_return_value $? "Failed to mkfs boot"
    else
        #ubuntu 24.04
        parted -s ${DEST_DISK} \
            resizepart $rootfs_partition "${rootfs_end}GB" \
            mkpart swap            linux-swap "${swap_start}GB"      "${tep_start}GB" \
            mkpart trusted_compute ext4       "${tep_start}GB"       "${reserved_start}GB" \
            mkpart reserved        ext4       "${reserved_start}GB"   "${persistent_start}GB" \
            mkpart persistent      ext4       "${persistent_start}GB" 100%

        check_return_value $? "Failed to create paritions"
    fi

    #swap space creation
    # mkswap "${DEST_DISK}${suffix}${swap_partition}"
    # check_return_value $? "Failed to mkswap"

    #TEP and reserved are not formated currently.
    #reserved
    mkfs -t ext4 -L reserved -F "${DEST_DISK}${suffix}${reserved_partition}"
    check_return_value $? "Failed to mkfs boot"
}

#####################################################################################
save_rootfs_on_ram(){
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
}

#####################################################################################
create_single_hdd_lvmg() {
    if [ $single_hdd -eq 0 ] && [ $lvm_size -gt 0 ];
    then
        cryptsetup luksFormat  \
            --batch-mode \
            --pbkdf-memory=2097152 \
            --pbkdf-parallel=8  \
            --cipher=aes-xts-plain64 \
            --reduce-device-size 32M \
            "${DEST_DISK}${suffix}${singlehdd_lvm_partition}" \
            $luks_key

        check_return_value $? "Failed to luks format for lvmvg ${DEST_DISK}${suffix}${singlehdd_lvm_partition}"

        cryptsetup luksOpen "${DEST_DISK}${suffix}${singlehdd_lvm_partition}" "lvmvg_crypt" --key-file=$luks_key
        check_return_value $? "Failed to luks open lvmvg_crypt"

        pvcreate "/dev/mapper/lvmvg_crypt"
        check_return_value $? "Failed to make mkfs ext4 on lvmvg_crypt"

        vgcreate lvmvg "/dev/mapper/lvmvg_crypt"
        check_return_value $? "Failed to create a lvmvg group"
        echo "vgcreate is completed"

        block_dev_actual_partition_uuid=$(blkid "${DEST_DISK}${suffix}${singlehdd_lvm_partition}" -s UUID -o value)
        echo -e "lvmvg_crypt UUID=${block_dev_actual_partition_uuid} none luks,discard,initramfs,keyscript=${initramfs_loc}/tpm2-cryptsetup" >> /mnt/etc/crypttab

        mkdir -p /mnt/media/lvmvg

        fstab_block_dev="/dev/mapper/lvmvg_crypt /media/lvmvg ext4 discard,errors=remount-ro       0 1"


        mount "/dev/mapper/lvmvg_crypt" /mnt/media/lvmvg
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
        #if its removable disk don't do LVM
        removable=$(lsblk -n -d -o RM "/dev/${block_dev}")
        if [ "$removable" -eq 1 ];
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

        # mkfs.ext4 -F "/dev/mapper/${block_dev}_crypt"
        pvcreate "/dev/mapper/${block_dev}_crypt"
        check_return_value $? "Failed to make mkfs ext4 on ${block_dev}_crypt"

        list_of_lvmg_part+=" /dev/mapper/${block_dev}_crypt"

        # add to fstab and crypttab

        block_dev_actual_partition_uuid=$(blkid "/dev/${block_dev}${part_suffix}1" -s UUID -o value)
        echo -e "${block_dev}_crypt UUID=${block_dev_actual_partition_uuid} none luks,discard,initramfs,keyscript=${initramfs_loc}/tpm2-cryptsetup" >> /mnt/etc/crypttab

        mkdir -p /mnt/media/${block_dev}
        # block_dev_uuid=$(blkid "/dev/mapper/${block_dev}_crypt" -s UUID -o value )
        fstab_block_dev="/dev/mapper/${block_dev}_crypt /media/${block_dev} ext4 discard,errors=remount-ro       0 1"
        #echo -e "${fstab_block_dev}" >> /mnt/etc/fstab

        mount "/dev/mapper/${block_dev}_crypt" /mnt/media/${block_dev}
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
mtab_to_fstab() {
    suffix=$(fix_partition_suffix)
    partprobe


    rootfs_uuid=$(blkid /dev/mapper/rootfs_crypt -s UUID -o value )
    swap_uuid=$(blkid /dev/mapper/swap_crypt -s UUID -o value )
    boot_uuid=$(blkid "${DEST_DISK}${suffix}${boot_partition}" -s UUID -o value)
    persistent_partition_uuid=$(blkid "${DEST_DISK}${suffix}${persistent_partition}" -s UUID -o value)
    
    # rootfs_uuid=$(lsblk /dev/mapper/rootfs_crypt -o uuid -n )
    # boot_uuid=$(lsblk "${DEST_DISK}${suffix}${boot_partition}" -o uuid -n )

    echo "rootfs_uuid ${rootfs_uuid} boot_uuid ${boot_uuid}"


    # fstab_swap_partition="${DEST_DISK}${suffix}${swap_partition} none swap sw 0 0"
    fstab_swap_partition="/dev/mapper/swap_crypt swap swap default 0 0"

    fstab_persistent_dev="/dev/mapper/persistent_crypt /var/lib/rancher ext4 discard,errors=remount-ro       0 1"

    fstab_complete="uuid=${rootfs_uuid} ${fstab_rootfs_partition}
${DEST_DISK}${suffix}${boot_partition} ${fstab_boot_partition}
${fstab_swap_partition}
${fstab_efi_partition}
${fstab_persistent_dev}"
    
    echo -e "${fstab_complete}" > /mnt/etc/fstab

    #update crypttab aswell
    rootfs_actual_partition_uuid=$(blkid "${DEST_DISK}${suffix}${rootfs_partition}" -s UUID -o value)
    echo -e "rootfs_crypt UUID=${rootfs_actual_partition_uuid} none luks,discard,keyscript=${initramfs_loc}/tpm2-cryptsetup" > /mnt/etc/crypttab

    swap_actual_partition_uuid=$(blkid "${DEST_DISK}${suffix}${swap_partition}" -s UUID -o value)
    echo -e "swap_crypt UUID=${swap_actual_partition_uuid} none luks,discard,keyscript=${initramfs_loc}/tpm2-cryptsetup" >> /mnt/etc/crypttab

    #update resume
    mkdir -p /mnt${initramfs_loc}/conf.d/
    echo -e "RESUME=/dev/mapper/swap_crypt" >/mnt${initramfs_loc}/conf.d/resume

    echo -e "persistent_crypt UUID=${persistent_partition_uuid} none luks,discard,initramfs,keyscript=${initramfs_loc}/tpm2-cryptsetup" >> /mnt/etc/crypttab

}

#####################################################################################
cleanup_rfs_backup() {
    # running this as part of another process to speed up the FDE
    dd if=/dev/zero of=${DEST_DISK}${suffix}${reserved_partition} bs=100MB count=20
}
#####################################################################################
enable_luks(){
    suffix=$(fix_partition_suffix)
    
    tpm2_getrandom 32 | xxd -p -c999 | tr -d '\n' > $luks_key
    check_return_value $? "Failed to get random number"
    # cat $luks_key

    cryptsetup luksFormat  \
            --batch-mode \
            --pbkdf-memory=2097152 \
            --pbkdf-parallel=8  \
            --cipher=aes-xts-plain64 \
            --reduce-device-size 32M \
            "${DEST_DISK}${suffix}${rootfs_partition}" \
            $luks_key
    
    check_return_value $? "Failed to luks format"

    cryptsetup luksOpen "${DEST_DISK}${suffix}${rootfs_partition}" rootfs_crypt --key-file=$luks_key
    check_return_value $? "Failed to luks open rootfs"

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

    # rm -rf rfs_backup/*
    # check_return_value $? "Failed to cleanup rfs backup"

    if [ "$ubuntu_version" == "22.04" ];
    then
    # in ubuntu 22.04 /boot is part of rootfs
        #move boot to different partition
        mkdir -p rfs/boot2
        cp -rp rfs/boot/* rfs/boot2
        check_return_value $? "Failed to backup boot"

        rm -rf rfs/boot
        mkdir rfs/boot

        mount "${DEST_DISK}${suffix}${boot_partition}" rfs/boot
        cp -rp rfs/boot2/* rfs/boot/
        check_return_value $? "Failed to copy the boot back"

        rm -rf rfs/boot2
        umount rfs/boot
    fi

    umount rfs
    umount rfs_backup

    #cleanup copied backup of rfs
    # cleanup_rfs_backup &

    ### setup swap luks
    cryptsetup luksFormat  \
        --batch-mode \
        --pbkdf-memory=2097152 \
        --pbkdf-parallel=8  \
        --cipher=aes-xts-plain64 \
        --reduce-device-size 32M \
        "${DEST_DISK}${suffix}${swap_partition}" \
        $luks_key
    
    check_return_value $? "Failed to luks format swap partition"

    cryptsetup luksOpen "${DEST_DISK}${suffix}${swap_partition}" swap_crypt --key-file=$luks_key
    check_return_value $? "Failed to luks open swap"

    #swap space creation
    mkswap /dev/mapper/swap_crypt
    check_return_value $? "Failed to mkswap"

    ### setup swap luks completed

    ### Setup persistent partition
    cryptsetup luksFormat  \
        --batch-mode \
        --pbkdf-memory=2097152 \
        --pbkdf-parallel=8  \
        --cipher=aes-xts-plain64 \
        --reduce-device-size 32M \
        "${DEST_DISK}${suffix}${persistent_partition}" \
        $luks_key

    check_return_value $? "Failed to luks format persistent partition"

    cryptsetup luksOpen "${DEST_DISK}${suffix}${persistent_partition}" persistent_crypt --key-file=$luks_key
    check_return_value $? "Failed to luks open persistent partition"

    mkfs.ext4 -F /dev/mapper/persistent_crypt
    check_return_value $? "Failed to make mkfs ext4 on persistent"

    

    ### Setup persistent partition completed

    # mounts needed to make the chroot work
    mount /dev/mapper/rootfs_crypt /mnt
    check_return_value $? "Failed to mount rootfs"
    
    mount "${DEST_DISK}${suffix}${boot_partition}" /mnt/boot
    check_return_value $? "Failed to mount /boot"
    
    mount "${DEST_DISK}${suffix}${efi_partition}" /mnt/boot/efi
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

    #install the required scripts inside the initramfs
    echo -e "${tpm2_cryptsetup}" > /mnt${initramfs_loc}/tpm2-cryptsetup
    get_tpm2_initramfs_tool > /mnt${initramfs_loc}/hooks/tpm2-initramfs-tool
    echo -e "${tpm2_pcrextend}" > /mnt${initramfs_loc}/scripts/init-bottom/pcr_extend.sh

    if [ "$ubuntu_version" == "24.04" ];
    then
        sed -i '$d' /mnt/etc/default/grub.d/40-force-partuuid.cfg
        kernel_version=$(ls /mnt/boot/vmlinuz-* | grep -o "[0-9]\+\.[0-9]\+\.\S*" | sort -V | tail -1)
    else
        kernel_version="all"
    fi

    # make them executable
    chmod +x /mnt${initramfs_loc}/tpm2-cryptsetup
    chmod +x /mnt${initramfs_loc}/hooks/tpm2-initramfs-tool
    chmod +x /mnt${initramfs_loc}/scripts/init-bottom/pcr_extend.sh

    mtab_to_fstab

    create_single_hdd_lvmg
    partition_other_devices

    rm /mnt/etc/resolv.conf
    touch /mnt/etc/resolv.conf
    mount --bind /etc/resolv.conf /mnt/etc/resolv.conf
    
    chroot /mnt /bin/bash <<EOT

    #inside installed ubuntu

    sed -i 's/#\$nrconf{kernelhints} = -1;/\$nrconf{kernelhints} = 0;/g' /etc/needrestart/needrestart.conf
    sed -i 's/#\$nrconf{ucodehints} = 0;/\$nrconf{ucodehints} = 0;/g' /etc/needrestart/needrestart.conf

    apt update

    dpkg --configure -a
    dpkg --triggers-only --pending

    apt install -y tpm2-tools cryptsetup tpm2-initramfs-tool

    dpkg --configure -a
    dpkg --triggers-only --pending
    #setup tpm
    tpm2-initramfs-tool seal --data $(cat /luks_key) --pcrs 15
    if [ $? -ne 0 ]
    then
	echo "tpm2-initramfs-tools failed"
	exit 1
    fi
    
    rm -rf /luks_key

    sleep 2
    echo "kernel_version $kernel_version"

    DPKG_MAINTSCRIPT_PACKAGE="" update-initramfs -u -k $kernel_version -v
    if [ $? -ne 0 ]
    then
	echo "update-initramfs failed"
	exit 1
    fi

    grub-install "${DEST_DISK}"
    if [ $? -ne 0 ]
    then
	echo "grub-install failed"
	exit 1
    fi

    update-grub
    if [ $? -ne 0 ]
    then
	echo "update grub failed"
	exit 1
    fi

    #TODO fix this as part of the deployment yaml
    sed -i 's/console=tty1 console=ttyS0/console=ttyS0,115200/' /boot/grub/grub.cfg

    systemctl mask sleep.target suspend.target hibernate.target hybrid-sleep.target systemd-logind
    
EOT



    #cleanup of mounts
    mount_points=($(grep -i "/mnt"  /proc/mounts | awk '{print $2}' | sort -nr))
    for mounted_dir in ${mount_points[@]};
    do
        umount $mounted_dir
    done
    # umount /mnt/sys
    # umount /mnt/proc      
    # umount /mnt/dev/pts    
    # umount /mnt/dev
    # umount /mnt/boot/efi
    # umount /mnt/boot
    # umount /mnt
}

#####################################################################################
main() {

    get_dest_disk
    set_correct_partition

    mininum_lvm_requested
    minimum_rootfs_requested

    is_single_hdd

    if [ $single_hdd -eq 0 ];
    then
        make_partition_single_hdd
    else
        make_partition
    fi

    save_rootfs_on_ram

    enable_luks
}


main
#####################################################################################
