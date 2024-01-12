<?php
$input_mac = filter_input(INPUT_GET, 'mac');
$input_boot_url = filter_input(INPUT_GET, 'boot_url');

function get_auto_ipxe($mac) {
  $data = array(
    'mac' => $mac,
  );
  $post_data = json_encode($data);
  $api_url = "http://$BOOTS_SERVICE_URL/auto.ipxe";

  $ch = curl_init($api_url);
  curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
  curl_setopt($ch, CURLOPT_HEADER, false);
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

  return $result;
}

$response = get_auto_ipxe($input_mac);
if ( empty($response) ){
  $custom_ipxe = <<<EOT
#!ipxe
echo Unable to get ipxe script for $input_mac. Retrying after 30 seconds
sleep 30
chain {$input_boot_url}/chain.php?mac=\${mac}&&boot_url={$input_boot_url}
EOT;
printf($custom_ipxe);
  }else{
printf($response);
  }
?>
