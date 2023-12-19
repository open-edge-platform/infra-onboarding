<?php
$input_mac = filter_input(INPUT_GET, 'mac');
$input_uuid = filter_input(INPUT_GET, 'uuid');
$input_serial_id = filter_input(INPUT_GET, 'serial_id');
$input_en_ip = filter_input(INPUT_GET, 'en_ip');
$input_boot_url = filter_input(INPUT_GET, 'boot_url');


function update_en_details($mac, $uuid, $serial_id, $ip) {

  global $input_boot_url;
  $data = array(
    'mac' => $mac,
    'uuid' => $uuid,
    'serial_id' => $serial_id,
    'ip' => $ip
  );
  $post_data = json_encode($data);

  
  $api_url = "http://{$input_boot_url}:8095/UpdateEN";

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
    exit(); // Unable to update details
  }else{
    echo "Write successful";
  }


?>
