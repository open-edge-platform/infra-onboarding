<?php
// Validate and sanitize inputs
$input_mac = filter_input(INPUT_GET, 'mac', FILTER_VALIDATE_MAC);
$input_boot_url = filter_input(INPUT_GET, 'boot_url', FILTER_VALIDATE_URL);

// Check and return error if any input is null/invalid and print which input is null/invalid
if (empty($input_mac) || empty($input_boot_url)) {
  $emptyInputs = [];
  if (empty($input_mac)) {
    $emptyInputs[] = 'mac';
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

function get_auto_ipxe($mac) {
  $api_url = "http://$BOOTS_SERVICE_URL/$mac/auto.ipxe";

  $ch = curl_init($api_url);
  curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
  curl_setopt($ch, CURLOPT_HEADER, false);
  curl_setopt($ch, CURLINFO_HEADER_OUT, true);
  curl_setopt($ch, CURLOPT_POST, true);
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
if (empty($response)) {
  $custom_ipxe = <<<EOT
#!ipxe
echo  Retrying until the workflow is created by the Onboarding Manager or done manually.
sleep 30
chain {$input_boot_url}/chain.php?mac=\${mac}&&boot_url={$input_boot_url}
EOT;
  printf($custom_ipxe);
} else {
  $response = filter_var($response, FILTER_SANITIZE_STRING);
  printf($response);
}
?>
