# Source catalog and evidence register

This catalog records the external and local sources used to shape the SOAR design. A
source is not treated as a product requirement automatically: normative standards are
translated into compatibility or control requirements, practitioner playbooks into
operational patterns, research into design hypotheses, and local operational material
into environment-specific constraints.

## Evidence classes

| Class | Meaning | How it is used |
|---|---|---|
| Normative | A published standard or government guidance with explicit requirements or recommended practices | Compatibility, interface, governance, or control baseline |
| Practitioner | A public operational playbook or vendor/project operational guide | Scenario steps, safety gates, and coordination patterns |
| Research | Peer-reviewed or preprint research with an explicit method or evaluation | Design hypotheses and test criteria; never a standalone control |
| Local | Existing KRTN operational guidance or project evidence | Local evidence-handling and lab constraints |
| Inference | A design decision synthesized from one or more sources | Must be validated with the owner and environment before production |

## Register

| ID | Class / authority | Version or date | Exact location reviewed | Extracted practice or finding | Design artifacts affected | Limitation |
|---|---|---|---|---|---|---|
| SRC-NIST-61R3 | Normative guidance / NIST | Apr 2025 | Executive Summary pp. 1-2; Table 1 p. 6; Community Profile Tables 2-3 pp. 11-35 | Incident response is integrated across all six CSF 2.0 Functions; Detect, Respond, and Recover cover discovery, prioritization, containment, eradication, recovery, reporting, and communications; Govern/Identify/Protect support preparation and lessons learned. | lifecycle, cases, recovery, metrics, governance, playbook phases | Voluntary guidance for most organizations; local legal and business requirements still control. |
| SRC-NIST-800184 | Practitioner guidance / NIST | Dec 2016 | Guide for Cybersecurity Event Recovery, recovery planning and recovery process | Recovery needs defined roles, prioritized services, communications, evidence-informed decisions, and exercises/lessons learned. | recovery playbook, backup/recovery, tabletop and closure criteria | Older guidance; use with current CSF 2.0 and organization-specific continuity plans. |
| SRC-CISA-IRVULN | Practitioner guidance / CISA, OMB, partners | Aug 2024 artifact | Incident Response Playbook checklist pp. 34-39; Vulnerability Response Playbook checklist pp. 50-53; appendices and roles/checklists | Use a common incident/vulnerability response sequence, define roles, preserve evidence, coordinate communications, track decisions, and close with lessons learned. | coordination, vulnerability playbook, evidence, communications, closure | Federal process details do not all apply outside government; adapt authority and notification obligations. |
| SRC-CISA-RANSOM | Practitioner guidance / CISA and MS-ISAC | Current web guide | Ransomware Guide, response checklist and preparation/containment/recovery recommendations | Prioritize preparation, isolation/containment, preservation, reporting/coordination, recovery from known-good backups, and post-incident improvement. | ransomware, evidence preservation, recovery, safety gates | Public guide is not a substitute for sector-specific legal, safety, or continuity procedures. |
| SRC-CISA-TTX | Practitioner guidance / CISA | Current web resource | Cybersecurity Scenarios and tabletop exercise resources | Exercise realistic scenarios with roles, injects, decisions, communications, and after-action improvement items. | tabletop tests, fixtures, governance, metrics | Exercise material must be adapted to the organization and must not touch production evidence. |
| SRC-CACAO-20 | Normative standard / OASIS | 2.0 CS01, Nov 2023 | §§1.1-1.3 pp. 8-11; §2.3 pp. 14-15; §3 pp. 18-30; §4 pp. 32-45; §5 pp. 46-63; §11 pp. 116-118 | A playbook has metadata, workflow, objects, markings and signatures; supports detection, investigation, mitigation, notification, prevention, and remediation; explicit start/end, action, parallel, condition, loop, versioning, and conformance are important. Shared playbooks require environmental adaptation. | playbook standard, execution semantics, versioning, schema, lifecycle | CACAO is an interchange model, not a complete provider connector contract or local approval policy. |
| SRC-OCSF | Normative schema framework / OCSF | Project current at review | OCSF overview: extensible vendor-agnostic core schema, taxonomy, attribute dictionary, JSON schema | Normalize security events to a vendor-agnostic model while preserving source and extension information. | event schema, normalization, source metadata, detection coverage | OCSF does not define the entire case, workflow, or authorization model. |
| SRC-TAXII-21 | Normative protocol / OASIS | 2.1 Standard, Jun 2021 | §§1.6.1-1.6.11 pp. 11-14; §§3.1-3.7 pp. 18-27; §§5.1-5.8 pp. 39-60; §8 pp. 64-66 | CTI exchange needs discoverable roots/endpoints, authentication, filtering, pagination, errors, object lifecycle, and certificate verification. | TI connector, error taxonomy, polling, pagination, audit | TAXII transports CTI; it does not determine provider verdict quality or local trust policy. |
| SRC-SIGMA-CORR | Normative detection format / Sigma | 2.1.0, Aug 2025 | Correlation specification: correlation section and types; `group-by`, `timespan`, `condition`, false positives, level | Correlation rules must state referenced rules, grouping scope, time window, threshold/condition, and known false positives; temporal ordering and backend behavior can create false positives/negatives. | detection engineering, correlation, risk scoring, tests | Backend translation and data quality still require environment-specific validation. |
| SRC-WAZUH-API | Practitioner technical documentation / Wazuh | Current docs | External API integration documentation | External alert integration must be authenticated, mapped, and treated as a provider-specific connector rather than leaking vendor fields into core logic. | Wazuh connector, ingestion, normalization, fixtures | Exact API behavior depends on Wazuh version and deployment. |
| SRC-SHUFFLE-WF | Practitioner technical documentation / Shuffle | Current docs | Workflow documentation | Workflow steps need explicit inputs/outputs, branching, connector actions, and observable execution state. | execution semantics, connector contract, replay | Project documentation is not a security control or interoperability standard. |
| SRC-SOAR-USE | Research / Bridges et al. | 2022 | Experimental design and findings in “Testing SOAR Tools in Use”; 24 participants and 6 commercial tools | SOAR improved efficiency and reduced context switching, but ticket accuracy/completeness decreased; senior analysts were concerned about overautomation; decision assistance was preferred. | HITL, analyst task, quality metrics, tabletop and replay tests | Small study and tool/configuration-specific; it motivates tests rather than proving universal effects. |
| SRC-OIIRP | Research / Cardiff University, Computers & Security | 2024 publication record | Abstract and paper “Operations-informed incident response playbooks” | Playbooks should connect response process with an operational model and critical thresholds so actions can be evaluated against business impact. | operational context, response decision matrix, ransomware, approval | Conceptual contribution and ransomware demonstration; thresholds must come from the business. |
| SRC-ICSECURE | Research / IC-SECURE | 2023 | “Intelligent System for Assisting Security Experts in Generating Playbooks for Automated Incident Response” | Treat playbook generation as expert-assisted investigative and response workflow design with structured actions/API modules. | playbook authoring, action catalog, AI safety, review workflow | Research prototype; generated steps require human review and provider validation. |
| SRC-APIRO | Research / Sworna, Islam, Babar | 2022 | Abstract and evaluation in “APIRO: A Framework for Automated Security Tools API Recommendation” | Security-tool API discovery suffers from heterogeneous documentation and semantic variation; automated recommendation is feasible but must be evaluated against tools and tasks. | connector catalog, capability matrix, API contract, connector tests | 3 tools and augmented data; not evidence that recommendations are safe to execute. |
| SRC-LOCAL-DFIR | Local operational guidance / KRTN DFIR workspace | Current repository | `../../dfir/AGENTS.md`; `../../dfir/playbooks/README.md`; Windows/Linux/Cloud/Network/Malware playbooks | Treat original evidence as read-only; write derived work to case work/exports; record hashes, custody, collector, timestamps, retention, collection order, prerequisites, validation, and stop conditions. | evidence model, custody, DFIR playbook, safe response, test boundaries | Local guidance is authoritative for this workspace, not a universal organizational policy. |

## Canonical URLs and local paths

The URLs below are the sources to retrieve again when a version changes. The catalog
was reviewed on 2026-08-30. The local source is intentionally a repository-relative
path rather than a web citation.

| ID | Canonical source |
|---|---|
| SRC-NIST-61R3 | [NIST SP 800-61 Rev. 3](https://csrc.nist.gov/pubs/sp/800/61/r3/final) and [official PDF](https://nvlpubs.nist.gov/nistpubs/SpecialPublications/NIST.SP.800-61r3.pdf) |
| SRC-NIST-800184 | [NIST SP 800-184 PDF](https://tsapps.nist.gov/publication/get_pdf.cfm?pub_id=922303) |
| SRC-CISA-IRVULN | [CISA Federal Incident and Vulnerability Response Playbooks PDF](https://www.cisa.gov/sites/default/files/2024-08/Federal_Government_Cybersecurity_Incident_and_Vulnerability_Response_Playbooks_508C.pdf) |
| SRC-CISA-RANSOM | [CISA StopRansomware Ransomware Guide](https://www.cisa.gov/stopransomware/ransomware-guide) |
| SRC-CISA-TTX | [CISA Cybersecurity Scenarios](https://www.cisa.gov/resources-tools/resources/cybersecurity-scenarios) |
| SRC-CACAO-20 | [OASIS CACAO Security Playbooks 2.0](https://docs.oasis-open.org/cacao/security-playbooks/v2.0/security-playbooks-v2.0.html) |
| SRC-OCSF | [Open Cybersecurity Schema Framework](https://ocsf.io/) |
| SRC-TAXII-21 | [OASIS TAXII 2.1](https://docs.oasis-open.org/cti/taxii/v2.1/taxii-v2.1.html) |
| SRC-SIGMA-CORR | [Sigma Correlation Rules Specification](https://sigmahq.io/sigma-specification/specification/sigma-correlation-rules-specification.html) |
| SRC-WAZUH-API | [Wazuh external API integration](https://documentation.wazuh.com/current/user-manual/manager/integration-with-external-apis.html) |
| SRC-SHUFFLE-WF | [Shuffle workflow documentation](https://github.com/Shuffle/shuffle-docs/blob/master/docs/workflows.md) |
| SRC-SOAR-USE | [Testing SOAR Tools in Use](https://arxiv.org/abs/2208.06075) |
| SRC-OIIRP | [Operations-informed incident response playbooks](https://orca.cardiff.ac.uk/id/eprint/165303/) |
| SRC-ICSECURE | [IC-SECURE](https://arxiv.org/abs/2311.03825) |
| SRC-APIRO | [APIRO](https://arxiv.org/abs/2201.07959) |
| SRC-LOCAL-DFIR | `../../dfir/AGENTS.md` and `../../dfir/playbooks/README.md` |

## Source-to-design rules

1. A source claim is accepted only when the URL, version/date, and reviewed section/page are recorded.
2. A design decision is not called “standard-compliant” unless the relevant normative source and conformance test are named.
3. A research finding becomes a hypothesis with a measurable test, not a hard-coded product behavior.
4. Every source-derived playbook carries its source IDs, adaptation notes, approval policy, stop conditions, evidence outputs, and tests.
5. Local deployment values such as thresholds, jurisdictions, owners, provider fields, and notification contacts remain configuration owned by the organization.
