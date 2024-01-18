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
# Patition information
rootfs_partition=1
boot_partition=2
swap_partition=3
tep_partition=4
reserved_partition=5
efi_partition=15

# DEST_DISK set from the template.yaml file as an environment variable.

#####################################################################################
# Partitions in %
# read as 90% or 91%
rootfs_space_end=90
boot_space_start=90
boot_space_end=91
swap_space_start=91

# Size in GBs
tep_size=1
reserved_size=5
boot_size=5

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
    #if there were any problems when the ubuntu was streamed.
    printf 'Fix\n' | parted ---pretend-input-tty -l

    list_block_devices=($(lsblk -o NAME,TYPE | grep -i disk  | awk  '$1 ~ /sd*|nvme*/ {print $1}'))
    for block_dev in ${list_block_devices[@]};
    do
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
make_partition() {

    #if there were any problems when the ubuntu was streamed.
    printf 'Fix\n' | parted ---pretend-input-tty -l

    #ram_size
    ram_size=$(free -g | grep -i mem | awk '{ print $2 }' )
    swap_size=$(( $ram_size + 2 ))

    total_size_disk=$(parted -s ${DEST_DISK} p | grep -i ${DEST_DISK} | awk '{ print $3 }' | sed  's/GB//g' )

    swap_space_end=$(echo $swap_size $total_size_disk $swap_space_start | awk '{print int(($1*100)/$2 + $3)}' )

    #tep space is added after swap space
    tep_space_end=$(( $swap_space_end + $tep_size ))

    echo "Dest_disk=${DEST_DISK} tep_space_end=$tep_space_end swap_space_end=$swap_space_end"

    #####
    reserved_start=$(( $total_size_disk - $reserved_size ))
    tep_start=$(( $reserved_start - $tep_size ))
    swap_start=$(( $tep_start - $swap_size ))
    boot_start=$(( $swap_start - $boot_size ))
    rootfs_end=$boot_start
    #####
    
    parted -s ${DEST_DISK} \
	   resizepart $rootfs_partition "${rootfs_end}GB" \
	   mkpart primary ext4 "${boot_start}GB" "${swap_start}GB" \
	   mkpart primary linux-swap "${swap_start}GB" "${tep_start}GB" \
	   mkpart primary ext4 "${tep_start}GB"  "${reserved_start}GB" \
	   mkpart primary ext4 "${reserved_start}GB"  100%

    check_return_value $? "Failed to create paritions"
    
    suffix=$(fix_partition_suffix)

    #/boot is now kept in a different partition
    mkfs -t ext4 -L boot -F "${DEST_DISK}${suffix}${boot_partition}"
    check_return_value $? "Failed to mkfs boot"

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
partition_other_devices() {
    list_block_devices=($(lsblk -o NAME,TYPE | grep -i disk  | awk  '$1 ~ /sd*|nvme*/ {print $1}'))
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
		   --reduce-device-size 32M \
		   "/dev/${block_dev}${part_suffix}1" \
		   $luks_key
	
	check_return_value $? "Failed to luks format for /dev/${block_dev}${part_suffix}1"

	cryptsetup luksOpen "/dev/${block_dev}${part_suffix}1" "${block_dev}_crypt" --key-file=$luks_key
	check_return_value $? "Failed to luks open ${block_dev}${part_suffix}1_crypt"

	mkfs.ext4 -F "/dev/mapper/${block_dev}_crypt"
	check_return_value $? "Failed to make mkfs ext4 on ${block_dev}_crypt"

	# add to fstab and crypttab
	
	block_dev_actual_partition_uuid=$(blkid "/dev/${block_dev}${part_suffix}1" -s UUID -o value)
	echo -e "${block_dev}_crypt UUID=${block_dev_actual_partition_uuid} none luks,discard,initramfs,keyscript=/etc/initramfs-tools/tpm2-cryptsetup" >> /mnt/etc/crypttab

	mkdir -p /mnt/media/${block_dev}
	# block_dev_uuid=$(blkid "/dev/mapper/${block_dev}_crypt" -s UUID -o value )
	fstab_block_dev="/dev/mapper/${block_dev}_crypt /media/${block_dev} ext4 discard,errors=remount-ro       0 1"
	echo -e "${fstab_block_dev}" >> /mnt/etc/fstab

	mount "/dev/mapper/${block_dev}_crypt" /mnt/media/${block_dev}
    done
    
}

#####################################################################################
mtab_to_fstab() {
    suffix=$(fix_partition_suffix)
    partprobe


    rootfs_uuid=$(blkid /dev/mapper/rootfs_crypt -s UUID -o value )
    swap_uuid=$(blkid /dev/mapper/swap_crypt -s UUID -o value )
    boot_uuid=$(blkid "${DEST_DISK}${suffix}${boot_partition}" -s UUID -o value)
    
    # rootfs_uuid=$(lsblk /dev/mapper/rootfs_crypt -o uuid -n )
    # boot_uuid=$(lsblk "${DEST_DISK}${suffix}${boot_partition}" -o uuid -n )

    echo "rootfs_uuid ${rootfs_uuid} boot_uuid ${boot_uuid}"


    # fstab_swap_partition="${DEST_DISK}${suffix}${swap_partition} none swap sw 0 0"
    fstab_swap_partition="/dev/mapper/swap_crypt swap swap default 0 0"

    fstab_complete="uuid=${rootfs_uuid} ${fstab_rootfs_partition}
${DEST_DISK}${suffix}${boot_partition} ${fstab_boot_partition}
${fstab_swap_partition}
${fstab_efi_partition}"
    
    echo -e "${fstab_complete}" > /mnt/etc/fstab

    #update crypttab aswell
    rootfs_actual_partition_uuid=$(blkid "${DEST_DISK}${suffix}${rootfs_partition}" -s UUID -o value)
    echo -e "rootfs_crypt UUID=${rootfs_actual_partition_uuid} none luks,discard,keyscript=/etc/initramfs-tools/tpm2-cryptsetup" > /mnt/etc/crypttab

    swap_actual_partition_uuid=$(blkid "${DEST_DISK}${suffix}${swap_partition}" -s UUID -o value)
    echo -e "swap_crypt UUID=${swap_actual_partition_uuid} none luks,discard,keyscript=/etc/initramfs-tools/tpm2-cryptsetup" >> /mnt/etc/crypttab

    #update resume
    echo -e "RESUME=/dev/mapper/swap_crypt" >/mnt/etc/initramfs-tools/conf.d/resume
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
    umount rfs
    umount rfs_backup

    #cleanup copied backup of rfs
    dd if=/dev/zero of=${DEST_DISK}${suffix}${reserved_partition} bs=100MB

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
    echo -e "${tpm2_cryptsetup}" > /mnt/etc/initramfs-tools/tpm2-cryptsetup
    echo -e "${tpm2_initramfs_tool}" > /mnt/etc/initramfs-tools/hooks/tpm2-initramfs-tool
    echo -e "${tpm2_pcrextend}" > /mnt/etc/initramfs-tools/scripts/init-bottom/pcr_extend.sh
    
    # make them executable
    chmod +x /mnt/etc/initramfs-tools/tpm2-cryptsetup
    chmod +x /mnt/etc/initramfs-tools/hooks/tpm2-initramfs-tool
    chmod +x /mnt/etc/initramfs-tools/scripts/init-bottom/pcr_extend.sh

    mtab_to_fstab

    partition_other_devices
    
    chroot /mnt /bin/bash <<EOT

    #inside installed ubuntu

    sed -i 's/#\$nrconf{kernelhints} = -1;/\$nrconf{kernelhints} = 0;/g' /etc/needrestart/needrestart.conf
    sed -i 's/#\$nrconf{ucodehints} = 0;/\$nrconf{ucodehints} = 0;/g' /etc/needrestart/needrestart.conf

    apt update

    apt install -y tpm2-tools cryptsetup tpm2-initramfs-tool

    #setup tpm
    tpm2-initramfs-tool seal --data $(cat /luks_key) --pcrs 15
    if [ $? -ne 0 ]
    then
	echo "tpm2-initramfs-tools failed"
	exit 1
    fi
    
    rm -rf /luks_key

    update-initramfs -u -k all
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
    # sed -i 's/console=tty1 console=ttyS0/console=ttyS0,115200/' /boot/grub/grub.cfg    
    
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
    make_partition
    save_rootfs_on_ram

    enable_luks
}


main
#####################################################################################
