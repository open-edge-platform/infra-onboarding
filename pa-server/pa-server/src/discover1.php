<?php
// Validate and sanitize inputs
$input_mac = filter_input(INPUT_GET, 'mac', FILTER_VALIDATE_MAC);
$input_uuid = filter_input(INPUT_GET, 'uuid', FILTER_SANITIZE_STRING);
$input_serial_id = filter_input(INPUT_GET, 'serial_id', FILTER_SANITIZE_STRING);
$input_en_ip = filter_input(INPUT_GET, 'en_ip', FILTER_VALIDATE_IP);
$input_boot_url = filter_input(INPUT_GET, 'boot_url', FILTER_VALIDATE_URL);

// Check and return error if any input is null/invalid and print which input is null/invalid
if (empty($input_mac) || empty($input_uuid) || empty($input_serial_id) || empty($input_en_ip) || empty($input_boot_url)) {
  $emptyInputs = [];
  if (empty($input_mac)) {
    $emptyInputs[] = 'mac';
  }
  if (empty($input_uuid)) {
    $emptyInputs[] = 'uuid';
  }
  if (empty($input_serial_id)) {
    $emptyInputs[] = 'serial_id';
  }
  if (empty($input_en_ip)) {
    $emptyInputs[] = 'en_ip';
  }
  if (empty($input_boot_url)) {
    $emptyInputs[] = 'boot_url';
  }
  $errorMessage = 'Error: Missing or invalid input parameters for ' . implode(', ', $emptyInputs);
  $custom_ipxe = <<<EOT
#!ipxe
echo $errorMessage
sleep 5
EOT;
printf($custom_ipxe);
  exit();
}
function update_en_details($mac, $uuid, $serial_id, $ip) {
  $data = array(
    'mac' => $mac,
    'uuid' => $uuid,
    'serial_id' => $serial_id,
    'ip' => $ip
  );
  $post_data = json_encode($data);
  $api_url = "http://$BOOTS_SERVICE_URL/updateEN";

  $ch = curl_init($api_url);
  curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
  curl_setopt($ch, CURLINFO_HEADER_OUT, true);
  curl_setopt($ch, CURLOPT_POST, true);
  curl_setopt($ch, CURLOPT_POSTFIELDS, $post_data);
  curl_setopt($ch, CURLOPT_TIMEOUT, 1800);
  curl_setopt($ch, CURLOPT_HTTPHEADER, array(
    'Content-Type: application/json')
  );

  $result = curl_exec($ch);
  if (curl_errno($ch)) {
    print_r("Curl error" . curl_error($ch));
  }
  curl_close($ch);
  $data = json_decode($result, true);
  return $data['status'];
}

$response = update_en_details($input_mac, $input_uuid, $input_serial_id, $input_en_ip);
if ( $response != "pass" ){
  $custom_ipxe = <<<EOT
#!ipxe
echo Unable to update inventory for $input_mac . Retrying after 30 seconds.
sleep 30
chain {$input_boot_url}/discover.php?mac=\${mac}&&uuid=\${uuid}&&serial_id=\${serial}&&en_ip=\${ip}&&boot_url={$input_boot_url}
EOT;
printf($custom_ipxe);
  }else{
  $custom_ipxe = <<<EOT
#!ipxe
echo Inventory is updated for $input_mac
echo Checking if workflow is available for $input_mac
chain {$input_boot_url}/chain.php?mac=\${mac}&&boot_url={$input_boot_url}
EOT;
printf($custom_ipxe);
  }
?>
