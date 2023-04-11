##### SPDX-FileCopyrightText: (C) 2022 Intel Corporation
##### SPDX-License-Identifier: LicenseRef-Intel
--------------------------------------------------------

Author Mandatory (to be filled by PR Author/Submitter)
------------------------------------------------------
- Developer who submits the Pull Request for merge is required to mark the checklist below as applicable for the PR changes submitted.  
- Those checklist items which are not marked are considered as not applicable for the PR change.  
- Items marked with an asterisk suffix are mandatory items to check and if not marked will be treated as non-compliant pull requests by the developers for Inner Source Development Model (ISDM) compliance

### PULL DESCRIPTION
_Provide a 1-2 line brief overview of the changes submitted through the Pull Request..._

### Impact Analysis

| Info | Please fill out this column |
| ------ | ----------- |
| Root Cause | Node details update was adding extra '-' in case of no variant |
| Jira ticket | Add the name to the Jira ticket eg: "LPEDN-622". Automation will do the linking to Jira |

### CODE MAINTAINABILITY
- [ ] **_Every commit is a single defect fix and does not mix feature addition or changes\*_**
- [ ] Added required new tests relevant to the changes
	- [ ] PR contains URL links to functional tests executed with the new tests 
- [ ] Updated Documentation as relevant to the changes
- [ ] PR change contains code related to security
- [ ] PR introduces changes that breaks compatibility with other modules/services (If YES, please provide description)
- [ ] Specific instructions or information for code reviewers (If any):

Maintainer Mandatory (to be filled by PR Reviewer/Approving Maintainer)
-----------------------------------------------------------------------
- Maintainer who approves the Pull Request for merge is required to mark the checklist below as appropriate for the PR change reviewed as key proof of attestation indicating reasons for merge. 
- Those checklist items which are not marked are considered as not applicable for the PR change. 
- Items marked with an asterisk suffix are mandatory items to check and if not marked will be treated as non-compliant pull requests by the maintainers for ISDM compliance.

### QUALITY CHECKS
- [ ] Architectural and Design Fit
- [ ] **_Quality of code (At least one should be checked as applicable)\*_**
	- [ ] Commit Message meets guidelines
	- [ ] PR changes adhere to industry practices and standards
	- [ ] Upstream expectations are met
	- [ ] Adopted domain specific coding standards 
	- [ ] Error and exception code paths implemented correctly
	- [ ] Code reviewed for domain or language specific anti-patterns
	- [ ] Code is adequately commented
	- [ ] Code copyright is correct
	- [ ] Tracing output are minimized and logic
	- [ ] Confusing logic is explained in comments
	- [ ] Commit comment can be used to design a new test case for the changes

### CODE REVIEW IMPACT
- Summary of Defects Detected in Code Review: **\<%P1*xx,P2*xx,P3*xx,P4*xx%>** \
Note P1/P2/P3/P4 denotes severity of defects found (Showstopper/High/Medium/Low) and xx denotes number of defects found

### SECURITY CHECKS
Please check if your PR fulfills the following requirements:

- [ ] Follow best practices when handling primitive data types
- [ ] Avoid unsafe functions
- [ ] Configure minimal permissions when opening pipes and ports
- [ ] All forms of input validated
- [ ] Error and exception handling implemented

# _Code must act as a teacher for future developers_
