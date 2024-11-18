#!/usr/bin/env bash
# SPDX-FileCopyrightText: (C) 2024 Intel Corporation
# SPDX-License-Identifier: LicenseRef-Intel

# Review each source code folder for fuzz tests and run them
homeDir=$(pwd)
anyTestFailed=0

for testFile in "${@}"; do
	fuzzTestCount=$(grep 'func Fuzz' "${testFile}" -c1)
	if [ "${fuzzTestCount}" -ne "0" ]; then
		echo "${fuzzTestCount}" fuzz tests found in "${testFile}"
		checkFuzzTest=$(grep 'func Fuzz' "${testFile}" | cut -d '(' -f 1 | cut -d ' ' -f 2)
		for fuzzTest in ${checkFuzzTest}; do    #
			fuzzFileName="${homeDir}/${testFile}"
			echo "running ${fuzzTest} test case"
			cd "$(dirname "${fuzzFileName}")" || exit
			logFile="${homeDir}/fuzz_${fuzzTest}.log"
			go test -fuzz "${fuzzTest}" -fuzztime 1m > "${logFile}" 2>&1
			exitStatus=$?
			echo "Output written to ${logFile}"
			cd "${homeDir}" || exit
			if [ $exitStatus -ne 0 ]; then
				echo "Fuzz test ${fuzzTest} in package ${fuzzFileName} FAILED"
				anyTestFailed=1
			fi			
		done
	fi
	echo
done

cd "${homeDir}" || exit
if [ $anyTestFailed -ne 0 ]; then
	echo "One or more fuzz tests failed."
	exit 1
else
	echo "All fuzz tests passed."
	exit 0
fi
# done