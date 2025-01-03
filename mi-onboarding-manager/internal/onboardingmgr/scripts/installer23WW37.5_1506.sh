#!/bin/bash
# Copyright (C) 2023 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

################################################################
#
# This script will Setup the Intel Proxies, setup the apt
# configuration & install all the Platform Software Packages
# with platform POR kernel
#
# usage: sudo ./installer.sh
#
# source: Configurable BKC Service
# shellcheck disable=all
set -e

current_workspace="$PWD"

ProxySetUp() {
##################################################################
##.............. Setting the Intel Proxies .......................

echo "$(date): Setting up Proxies..."
sed -i 's/#WaylandEnable=/WaylandEnable=/g' /etc/gdm3/custom.conf;
sed -i 's/"1"/"0"/g' /etc/apt/apt.conf.d/20auto-upgrades;
echo 'source /etc/profile.d/mesa_driver.sh' | sudo tee -a /etc/bash.bashrc;
echo 'set enable-bracketed-paste off' >> /etc/inputrc;
echo 'sys_olvtelemetry ALL=(ALL) NOPASSWD: /usr/sbin/biosdecode, /usr/sbin/dmidecode, /usr/sbin/ownership, /usr/sbin/vpddecode' > /etc/sudoers.d/user-sudo;
echo 'user ALL=(ALL) NOPASSWD: ALL' >> /etc/sudoers.d/user-sudo;
chmod 440 /etc/sudoers.d/user-sudo;
{
  echo 'http_proxy=http://proxy-dmz.intel.com:911'
  echo 'https_proxy=http://proxy-dmz.intel.com:912'
  echo 'ftp_proxy=http://proxy-dmz.intel.com:911'
  echo 'socks_server=http://proxy-dmz.intel.com:1080'
  echo 'no_proxy=localhost,*.intel.com,intel.com,127.0.0.1' ;
} > /etc/environment
sed -i 's/#  AutomaticLoginEnable =/AutomaticLoginEnable =/g' /etc/gdm3/custom.conf;
sed -i 's/#  AutomaticLogin = user1/AutomaticLogin = user/g' /etc/gdm3/custom.conf;
echo 'kernel.printk = 7 4 1 7' > /etc/sysctl.d/99-kernel-printk.conf;
echo 'kernel.dmesg_restrict = 0' >> /etc/sysctl.d/99-kernel-printk.conf;
./etc/environment;
export http_proxy https_proxy ftp_proxy socks_server no_proxy;
apt list --installed > /opt/Bom-list.txt;
{
  echo 'BUILD_TIME='$(date +%Y%m%d-%H%M) 
  echo 'PLATFORM=RPL-P'
} > /opt/jenkins-build-timestamp;

echo -e 'ADL KERNEL=5.15.96-lts-230421t211918z
RPL KERNEL=5.19-intel' >> /opt/jenkins-build-timestamp;
echo -e 'RPL EDGE KERNEL=5.19.0-mainline-tracking-eb-230725t100749z
ADL EDGE KERNEL=5.15.96-lts-230421t211918z' >> /opt/jenkins-build-timestamp;
echo -e 'RPL-PS KERNEL=5.19-intel' >> /opt/jenkins-build-timestamp
}


PPAUpdate() {
##################################################################
##.............. Setting the Intel Proxies .......................
echo "$(date): Adding PPA & GPG Key..."
echo 'Acquire::ftp::Proxy "http://proxy-dmz.intel.com:911";' > /etc/apt/apt.conf.d/99proxy.conf;
echo 'Acquire::http::Proxy "http://proxy-dmz.intel.com:911";' >> /etc/apt/apt.conf.d/99proxy.conf;
echo 'Acquire::https::Proxy "http://proxy-dmz.intel.com:911";' >> /etc/apt/apt.conf.d/99proxy.conf;
echo 'Acquire::https::proxy::apt.repos.intel.com "http://proxy-dmz.intel.com:911";' >> /etc/apt/apt.conf.d/99proxy.conf;
echo 'Acquire::https::proxy::af01p-png.devtools.intel.com "DIRECT";' >> /etc/apt/apt.conf.d/99proxy.conf;
echo 'Acquire::https::proxy::ubit-artifactory-or.intel.com "DIRECT";' >> /etc/apt/apt.conf.d/99proxy.conf;
echo 'Acquire::https::proxy::*.intel.com "DIRECT";' >> /etc/apt/apt.conf.d/99proxy.conf;
mkdir -p /etc/systemd/system/docker.service.d;
echo -e '[Service]
Environment="HTTP_PROXY=http://proxy-dmz.intel.com:911"
Environment="HTTPS_PROXY=http://proxy-dmz.intel.com:911"
Environment="NO_PROXY=amr-registry-pre.caas.intel.com,gar-registry.caas.intel.com,10.49.76.0/24"' >> /etc/systemd/system/docker.service.d/http-proxy.conf;
wget https://af01p-png.devtools.intel.com/artifactory/hspe-edge-png-local/ubuntu/keys/adl-hirsute-public.gpg -O /etc/apt/trusted.gpg.d/adl-hirsute-public.gpg;
env https_proxy=http://proxy-dmz.intel.com:911 wget https://download.01.org/intel-linux-overlay/ubuntu/E6FA98203588250569758E97D176E3162086EE4C.gpg -O /etc/apt/trusted.gpg.d/E6FA98203588250569758E97D176E3162086EE4C.gpg;
wget https://af01p-png.devtools.intel.com/artifactory/hspe-edge-repos-png-local-png-local/ubuntu/pub.gpg -O /etc/apt/trusted.gpg.d/hspe-edge-repos-png-local-png-local.gpg;
echo deb https://download.01.org/intel-linux-overlay/ubuntu/ jammy main non-free multimedia > /etc/apt/sources.list.d/intel-internal.list;
echo 'deb [trusted=yes] https://ubit-artifactory-or.intel.com/artifactory/turtle-creek-debian-local jammy universe' > /etc/apt/sources.list.d/inb.list;
echo -e "Package: *
Pin: origin download.01.org
Pin-Priority: 1001" >> /etc/apt/preferences.d/priorities;
echo deb http://archive.ubuntu.com/ubuntu/ jammy-proposed main universe restricted > /etc/apt/sources.list.d/proposed_fixup.list;
cat /etc/apt/preferences.d/priorities;
grep -rn . /etc/apt/sources.list /etc/apt/sources.list.d/;
apt update;
echo 'N' | apt upgrade -y;
apt-get install -yq curl;
env https_proxy=http://proxy-dmz.intel.com:911 curl https://apt.repos.intel.com/intel-gpg-keys/GPG-PUB-KEY-INTEL-SW-PRODUCTS.PUB | gpg --dearmor > /etc/apt/trusted.gpg.d/GPG-PUB-KEY-INTEL-SW-PRODUCTS.gpg;
useradd --system -m -p jaiZ6dai -U rbfadmin;
useradd --system -m -U sys_olvtelemetry;
mkdir -m 700 -p ~sys_olvtelemetry/.ssh;
echo 'ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOPEVYF28+I92b3HFHOSlPQXt3kHXQ9IqtxFE4/0YkK5 swsbalabuser@BA02RNL99999' > ~sys_olvtelemetry/.ssh/authorized_keys;
echo 'ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDb2P8gBvsy9DkzC1WiXfvisMFf7PQvtdvVC4n22ot4D5KOVxgoaCnjZM6qAZ2AdWPBebxInnUeMvw0u6RjRnflpYtNPgN4qiE313j62CmD80f/N+jvIxmoGhgsGE4RAMFXQ6pNaB/8KblrpmWQ5VfEIt7JcSR3Qvnkl9I2bljJU9zrMieE+Nras7hstg8fVWtGNjQjJpMWmt1YGxVbQiea0jDBqpru6TqnOYGD48JdR8QzHq++xL82I3x8kPz6annAvCDSVmiw9Mz0YtAsPIDZj4ABm866a8/U2mKVUncXYrBG1/pHBJMDJeX3ggd/UK2NvU8uEDJmITXUZRP8kBaO7b2LnRO08+Pr+nvmwukCP/wXflfS59h7kXCo8+Xjx/PEMO4OyFYHQunOUf/XTC13iig/MLY0EbqU6D+Lg1N13eJocRSta50zV+m+/PG23Zd3/6UH0noxYezQV3dQmsstzKKXbm8vkBmdqCZEvEnFSgl0VmX5HpzZLYI3L3hBH8/wgiWinrs7K13pZ8+lXN0ZhhJhdo61juiYwy1gbHP0ihqGkePw7w0DSCu5s9fA7xDTy2YTjkMsKaT8rbTYG5hunokNswdOCNYJyiCF3zJ08Z5hlDqSJJOPRdjL3YTIr6QlWSea/pTjkWmmE7Mv8M15c4V8Y77x6DsTFWlmGQbf1Q== swsbalabuser@BA02RNL99999' >> ~sys_olvtelemetry/.ssh/authorized_keys;
chmod 600 ~sys_olvtelemetry/.ssh/authorized_keys;
chown sys_olvtelemetry:sys_olvtelemetry -R ~sys_olvtelemetry/.ssh;
sed -e 's@^GRUB_TIMEOUT_STYLE=hidden@# GRUB_TIMEOUT_STYLE=hidden@' -e 's@^GRUB_TIMEOUT=0@GRUB_TIMEOUT=5@g' -i /etc/default/grub
}

InstallPackage(){
##################################################################
echo "$(date): Installing Packages...................."
package=("vim,ocl-icd-libopencl1,curl,openssh-server,net-tools,automake,libtool,cmake,g++,gcc,git,build-essential,apt-transport-https,default-jre,docker-compose,gir1.2-gst-plugins-bad-1.0,gir1.2-gst-plugins-base-1.0,gir1.2-gstreamer-1.0,gir1.2-gst-rtsp-server-1.0,gstreamer1.0-alsa,gstreamer1.0-gl,gstreamer1.0-gtk3,gstreamer1.0-opencv,gstreamer1.0-plugins-bad,gstreamer1.0-plugins-bad-apps,gstreamer1.0-plugins-base,gstreamer1.0-plugins-base-apps,gstreamer1.0-plugins-good,gstreamer1.0-plugins-ugly,gstreamer1.0-pulseaudio,gstreamer1.0-qt5,gstreamer1.0-rtsp,gstreamer1.0-tools,gstreamer1.0-vaapi,gstreamer1.0-wpe,gstreamer1.0-x,inbc-program,inbm-cloudadapter-agent,inbm-configuration-agent,inbm-diagnostic-agent,inbm-dispatcher-agent,inbm-telemetry-agent,mqtt,tpm-provision,trtl,intel-media-va-driver-non-free,itt-dev,itt-staticdev,jhi,jhi-tests,libmfx1,libmfx-dev,libmfx-tools,libd3dadapter9-mesa,libd3dadapter9-mesa-dev,libdrm-amdgpu1,libdrm-common,libdrm-dev,libdrm-intel1,libdrm-nouveau2,libdrm-radeon1,libdrm-tests,libdrm2,libegl-mesa0,libegl1-mesa,libegl1-mesa-dev,libgbm-dev,libgbm1,libgl1-mesa-dev,libgl1-mesa-dri,libgl1-mesa-glx,libglapi-mesa,libgles2-mesa,libgles2-mesa-dev,libglx-mesa0,libgstrtspserver-1.0-dev,libgstrtspserver-1.0-0,libgstreamer-gl1.0-0,libgstreamer-opencv1.0-0,libgstreamer-plugins-bad1.0-0,libgstreamer-plugins-bad1.0-dev,libgstreamer-plugins-base1.0-0,libgstreamer-plugins-base1.0-dev,libgstreamer-plugins-good1.0-0,libgstreamer-plugins-good1.0-dev,libgstreamer1.0-0,libgstreamer1.0-dev,libigdgmm-dev,libigdgmm12,libigfxcmrt-dev,libigfxcmrt7,libmfx-gen1.2,libosmesa6,libosmesa6-dev,libtpms-dev,libtpms0,libva-dev,libva-drm2,libva-glx2,libva-wayland2,libva-x11-2,libva2,libwayland-bin,libwayland-client0,libwayland-cursor0,libwayland-dev,libwayland-doc,libwayland-egl-backend-dev,libwayland-egl1,libwayland-egl1-mesa,libwayland-server0,libweston-9-0,libweston-9-dev,libxatracker-dev,libxatracker2,linux-firmware,mesa-common-dev,mesa-utils,mesa-va-drivers,mesa-vdpau-drivers,mesa-vulkan-drivers,libvpl-dev,libmfx-gen-dev,onevpl-tools,ovmf,ovmf-ia32,qemu,qemu-efi,qemu-block-extra,qemu-guest-agent,qemu-system,qemu-system-arm,qemu-system-common,qemu-system-data,qemu-system-gui,qemu-system-mips,qemu-system-misc,qemu-system-ppc,qemu-system-s390x,qemu-system-sparc,qemu-system-x86,qemu-system-x86-microvm,qemu-user,qemu-user-binfmt,qemu-utils,va-driver-all,vainfo,weston,xserver-xorg-core,libvirt0,libvirt-clients,libvirt-daemon,libvirt-daemon-config-network,libvirt-daemon-config-nwfilter,libvirt-daemon-driver-lxc,libvirt-daemon-driver-qemu,libvirt-daemon-driver-storage-gluster,libvirt-daemon-driver-storage-iscsi-direct,libvirt-daemon-driver-storage-rbd,libvirt-daemon-driver-storage-zfs,libvirt-daemon-driver-vbox,libvirt-daemon-driver-xen,libvirt-daemon-system,libvirt-daemon-system-systemd,libvirt-dev,libvirt-doc,libvirt-login-shell,libvirt-sanlock,libvirt-wireshark,libnss-libvirt,swtpm,swtpm-tools,bmap-tools,adb,intel-gpu-tools,libssl3,libssl-dev,make,mosquitto,mosquitto-clients,ffmpeg,git-lfs,gnuplot,lbzip2,libglew-dev,libglm-dev,libsdl2-dev,mc,openssl,pciutils,python3-pandas,python3-pip,python3-seaborn,terminator,vim,wmctrl,wayland-protocols,gdbserver,ethtool,iperf3,msr-tools,powertop,linuxptp,lsscsi,tpm2-tools,tpm2-abrmd,binutils,cifs-utils,i2c-tools,xdotool,gnupg,lsb-release,ethtool,iproute2,socat,stress-ng")
IFS=',' read -ra package_list <<< "$package"
#apt-get update > package_install.log 2>&1
apt-get update;
for item in "${package_list[@]}";
do
    echo "Installing $item"
    #sudo apt-get install $item -y >> package_install.log 2>&1
    sudo apt-get install "$item" -y
done
echo "$(date): Packages Successfully Installed...................."

if [ ! -L "/usr/local/bin/cpupower" ]; then
  ln -s /usr/lib/linux-tools-5.15.0-*/cpupower /usr/local/bin/cpupower
  echo "Symbolic link created"
else
  echo "Symbolic link already exists"
fi

sudo usermod -aG docker "$USER"
sudo systemctl restart docker
sudo systemctl daemon-reload
sudo chmod 666 /var/run/docker.sock
#sudo apt-get install wget vim > /dev/null 2>&1
}

KernelUpdate() {
##################################################################
##.............. Setting the Intel Proxies .......................

echo "$(date): Updating Kernel.................."
#kernel Overlays
kernel_overlays="http://oak-07.jf.intel.com/ikt_kernel_deb_repo/pool/main/l/linux-5.15.96-lts-230421t211918z/linux-headers-5.15.96-lts-230421t211918z_5.15.96-184_amd64.deb,http://oak-07.jf.intel.com/ikt_kernel_deb_repo/pool/main/l/linux-5.15.96-lts-230421t211918z/linux-image-5.15.96-lts-230421t211918z_5.15.96-184_amd64.deb"
IFS=',' read -ra kernel_overlays_list <<< "$kernel_overlays"

# Download & Install the Kernel
for item in "${kernel_overlays_list[@]}"; do
    a=("${item//// }")
    echo "$(date): Downloading ${a[-1]}"
    wget -c --tries=3 --read-timeout=60 "$item" -q
    if [ $? -eq 0 ]; then
    	echo "Download successful!"
	echo "$(date): Installing ${a[-1]}"
    	sudo dpkg -i "$(echo "$item" | rev | cut -d'/' -f1 | rev)"
    	echo "$(date): Installed ${a[-1]}"
    	sudo rm "$(echo "$item" | rev | cut -d'/' -f1 | rev)"
    else
    	echo "Download failed. Exit status: $?"
	exit 1
    fi
done
#dpkg -i *.deb
echo "All Kernel debs are installed..."
#Update grub menu entry
#Replace default Boot kernel with Kernel version
kernel_version_string=$(echo "${kernel_overlays_list[0]}" | rev | cut -d'/' -f2 | rev)
kernel_version=$(echo "$kernel_version_string" | sed 's/linux-//g')
sudo sed -i 's/GRUB_DEFAULT=.*/GRUB_DEFAULT="Advanced options for Ubuntu>Ubuntu, with Linux '"$kernel_version"'"/' /etc/default/grub
sudo sed -i 's/GRUB_CMDLINE_LINUX=.*/GRUB_CMDLINE_LINUX="i915.enable_guc=7 i915.force_probe=* udmabuf.list_limit=8192 console=tty0 console=ttyS0,115200n8"/' /etc/default/grub
#Update GRUB
sudo update-grub
#sudo reboot
}

disk_partitioning() {
##################################################################
##............. Externding Disk Partitioning size.................
echo "$(date): Extentending DISK size & Mounting Partition................"
drive=$(lsblk -no pkname "$(findmnt -n / | awk '{ print $2 }')")
echo -e "n\n\n\n\nw" | sudo fdisk /dev/"$drive"
new_partition=$(lsblk -npo name /dev/"$drive" | tail -n1)
trimmed_partition=${new_partition#*/}
sudo mkfs.ext4 /"$trimmed_partition"
sudo mkdir -p /residues
sudo mount /"$trimmed_partition" /residues
echo "/$trimmed_partition     /residues       auto    defaults  0  1" | sudo tee -a /etc/fstab
echo "$(date): Mounting Successfully Done................"
}


#ProxySetUp
PPAUpdate 2>&1 | tee -a "$current_workspace/cbkc_output.log"
InstallPackage 2>&1 | tee -a "$current_workspace/cbkc_output.log"
disk_partitioning 2>&1 | tee -a "$current_workspace/cbkc_output.log"
KernelUpdate 2>&1 | tee -a "$current_workspace/cbkc_output.log"
ProxySetUp 2>&1 | tee -a "$current_workspace/cbkc_output.log"

echo "Rebooting Device now" | tee -a "$current_workspace/cbkc_output.log"
echo "Installation done" > "$current_workspace"/.base_pkg_install_done
sudo reboot
