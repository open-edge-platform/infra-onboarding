#!/bin/bash

OUTPUT_FILE="result.txt"
touch "$OUTPUT_FILE"
echo "created file:$CLIENT_MAC"
echo "macid: $CLIENT_MAC" > "$OUTPUT_FILE"
echo "rootfspart: $ROOTFS_PART" >> "$OUTPUT_FILE"
echo "loadbalacerip: $TINKER_HOST_IP" >> "$OUTPUT_FILE"
echo "rootfspartno: $ROOTFS_PARTNO" >> "$OUTPUT_FILE"
echo "clientimg: $CLIENT_IMG" >> "$OUTPUT_FILE"
echo "img: $CLIENT_IMG_TYPE" >> "$OUTPUT_FILE"
