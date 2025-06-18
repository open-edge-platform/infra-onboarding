# Full Disk Encryption (FDE) and Device Mapper Verity (DM-Verity)

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

---

slug: enable-security-features

name: enable-security-features

description: "This action sets up full disk encryption using LUKS to protect data on disk and enables DM-Verity(DMV) to
ensure the root filesystem hasn't been tampered with. It uses TPM for key generation and secure storage, and
configures encrypted partitions(optional) with integrity verification for enhanced system security."

version: 1.18.1

| env var | data type | default value | required | description |
|---------|-----------|---------------|----------|-------------|
| ENABLE_ONLY_DMVERITY | bool | true | yes |  When set to `true`, only DM-Verity is enabled. Set to `false` FDE, Secure Boot and DM-Verity is enabled. |


The below example will enable FDE, DM-V and Secure Boot on the target Edge Node.

```yaml
actions:
    - name: "enable-security-features"
        image:  registry-rs.edgeorchestration.intel.com/edge-orch/infra/tinker-actions/fde_dmv:1.17.2
        timeout: 560
        environment:
          ENABLE_ONLY_DMVERITY: false
```

The below example will enable  DM-V on a Edge Node.
If block disks of smaller size(32-110GB) is available then smaller partitioning scheme is used.
Applicable only for DM-verity.

```yaml
actions:
    - name: "enable-security-features"
        image:  registry-rs.edgeorchestration.intel.com/edge-orch/infra/tinker-actions/fde_dmv:1.17.2
        timeout: 560
        environment:
          ENABLE_ONLY_DMVERITY: true
```


This document explains Full Disk Encryption (FDE) and Device Mapper Verity (DM-Verity) implimentation,
along with the key differences between the two mechanisms and partition scheme.

---

## Requirements

### System Requirements

#### Hardware

- A Trusted Platform Module (TPM) device for key sealing and secure boot.
- A system with a single or multiple hard disk drives (HDDs) or NVMe drives.
- Minimum disk size required: 128GB.
- If disk size is lesser than 128GB user can define PARTITIONING_SCHEME="small" and continue with
  the provisioning process for only DM-Verity.

#### Software

- Linux-based operating system.
- Required packages: `cryptsetup`, `tpm2-tools`, `tpm2-initramfs-tool`, `grub`.

---

## Full Disk Encryption (FDE)

Full Disk Encryption (FDE) encrypts the entire disk to ensure that data is inaccessible without the encryption key.
This protects sensitive data even if the physical disk is stolen or removed.

---

## Device Mapper Verity (DM-Verity)

DM-Verity is a kernel feature that ensures the integrity of the root filesystem by validating data blocks against a
hash tree. It prevents unauthorized modifications to the filesystem, making it ideal for read-only root filesystems.

---

## Partition Scheme

### Partitions Overview

1. **Boot Partition**
   - **Purpose**: Stores the bootloader and kernel files required to boot the system.
   - **Minimum Size**: Not explicitly defined.

2. **RootFS Partition**
   - **Purpose**: Contains the system's root filesystem (essential system files and libraries).
   - **Minimum Size**: 3584 MB (3.5 GB).
   - **Actual Size**: 3584 MB (3.5 GB).

3. **EMT Persistent Partition**
   - **Purpose**: Stores persistent data specific to the Edge Microvisor Toolkit.
   - **Minimum Size**: Not explicitly defined.
   - **Actual Size**: Depends on available space (calculated dynamically).

4. **RootFS Hashmap Partition A**
   - **Purpose**: Stores hashmaps for verifying the integrity of RootFS during upgrades or normal operations (Part A).
   - **Minimum Size**: 100 MB.
   - **Actual Size**: 100 MB.

5. **RootFS Hashmap Partition B**
   - **Purpose**: Stores hashmaps for verifying the integrity of RootFS during upgrades or normal operations (Part B).
   - **Minimum Size**: 100 MB.
   - **Actual Size**: 100 MB.

6. **RootFS B Partition**
   - **Purpose**: Acts as a backup RootFS for A/B upgrades, enabling system recovery or OTA updates.
   - **Minimum Size**: 3584 MB (3.5 GB).
   - **Actual Size**: 3584 MB (3.5 GB).

7. **RootFS Roothash Partition**
   - **Purpose**: Stores cryptographic hash values of the root filesystem for verification purposes.
   - **Minimum Size**: 50 MB.
   - **Actual Size**: 50 MB.

8. **Swap Partition**
   - **Purpose**: Acts as virtual memory when the system runs out of physical RAM.
   - **Minimum Size**: Square root of RAM size (in GB).
   - **Actual Size**: Half of RAM size.

9. **Trusted Compute Partition**
   - **Purpose**: Stores temporary execution files or data.
   - **Minimum Size**: 14336 MB (14 GB).
   - **Actual Size**: 14336 MB (14 GB).

10. **Reserved Partition**
    - **Purpose**: Reserved for platform-specific or recovery purposes.
    - **Minimum Size**: 5120 MB (5 GB).
    - **Actual Size**: 5120 MB (5 GB).

11. **LVM Partition**
    - **Purpose**: Used for Logical Volume Management, allowing flexible disk space allocation.
    - **Minimum Size**: Remaining disk space after other partitions.
    - **Actual Size**: Remaining disk space or 100% if only one disk is present.

---

## High-Level Provisioning Sequence

### Initialization

1. **Set Global Variables and Flags**
   - Initialize variables like `COMPLETE_FDE_DMVERITY`.
   - Define functions for operations such as disk partitioning, encryption, and filesystem setup.

### Disk and Partition Setup

1. **Get Destination Disk (`get_dest_disk`)**
   - Identify the disk (`DEST_DISK`) to be used for provisioning.

2. **Check for Single HDD (`is_single_hdd`)**
   - Determine if the system has a single hard drive and adjust configurations accordingly.

3. **Create Partitions (`make_partition`)**
   - Partition the disk based on pre-defined sizes and flags.
   - Convert partition sizes from MB to sectors for compatibility with the `parted` command.

### Backup of RootFS

1. **Backup of Root Filesystem**
   - Save the root filesystem for later restoration after LUKS setup.

### Key Generation

1. **Generate LUKS Key**
   - Generate a 32-byte random key using the TPM's random number generator (`tpm2_getrandom`).
   - Store the key temporarily in the `luks2_key` file for encryption purposes.

### Encryption Setup

1. **Enable LUKS Encryption (`enable_luks`)**
   - Set up LUKS2 encryption on partitions such as rootfs, swap, and persistent partitions.
   - Handle encryption for DM-verity-related partitions if enabled.

2. **Seal passphrase to TPM**
   - Seal the LUKS passphrase to the TPM using `tpm2-initramfs-tool` for secure storage.
   - Securely delete passphrase from disk after sealing.

### Logical Volume Management

1. **Create Single HDD LVM Group (`create_single_hdd_lvmg`)**
   - Create an encrypted Logical Volume Group (LVM) if a single HDD is detected.

2. **Partition Other Devices (`partition_other_devices`)**
   - Partition and encrypt additional block devices if present.

### Verification and Finalization

1. **Update LUKS UUID (`update_luks_uuid`)**
   - Update the UUID for the encrypted root filesystem.

2. **Enable DM-Verity (`luks_format_verity_part`)**
   - If `COMPLETE_FDE_DMVERITY` is enabled, set up DM-verity partitions for integrity verification.

### Cleanup

1. **Unmount Filesystems**
   - Ensure all mounted filesystems are properly unmounted after provisioning.

2. **Cleanup Temporary Files**
   - Remove temporary data, such as backups stored in the reserved partition.

---

## Key Differences Between FDE and DMV Partitioning

### FDE Partitioning

- The `rootfs_partition` is fully encrypted to ensure data confidentiality.
- The `swap_partition` is also encrypted to protect sensitive data in virtual memory.
- The `boot_partition` and `efi_partition` remain unencrypted for compatibility with bootloaders.
- The `emt_persistent_partition` and `singlehdd_lvm_partition` are also encrypted but are used for specific purposes
 like persistent data storage.
- The `trusted_compute_partition` and `reserved_partition` are not encrypted but are used for Trusted Compute-based
 activities and recovery operations, respectively.

### DMV Partitioning

- Adds integrity verification to the `rootfs_partition` using hash maps stored in `root_hashmap_a_partition` and
 `root_hashmap_b_partition`.
- The `roothash_partition` is used to store the root hash for verifying the integrity of the root filesystem.
- The `boot_partition` and `efi_partition` remain unencrypted, similar to FDE, but the integrity of the boot environment
 is validated using TPM and PCR values.
- The `emt_persistent_partition` and other partitions are used similarly to FDE but with added integrity checks where
 applicable.

In summary, FDE focuses on encrypting partitions to ensure data confidentiality, while DMV adds integrity verification
 mechanisms to ensure that the root filesystem and other critical partitions remain unmodified and secure.
