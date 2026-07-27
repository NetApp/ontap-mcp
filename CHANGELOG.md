# Change Log
## [Repository](https://github.com/NetApp/ontap-mcp)

## 26.07.0 / 2026-07-28 Release

The ONTAP-MCP team is happy to announce that we've released the 26.07.0 version of ONTAP-MCP. 🤘

- :medal_sports: The ONTAP-MCP server now supports OAuth authentication to restrict any unauthorized access to your MCP server. Thanks @jbnetapp for raising. Configuration details:  https://netapp.github.io/ontap-mcp/26.07/mcp-oauth/

- :medal_sports: The ONTAP-MCP server supports serving over HTTPS (TLS) for remote connections to your MCP server. Thanks @werenzo for raising. Configuration details: https://netapp.github.io/ontap-mcp/26.07/prepare-ontap/#serving-over-https-tls

- :medal_sports: The ONTAP-MCP server includes an Helm chart. Thanks @ReBaunana for contributing. Deployment details: https://netapp.github.io/ontap-mcp/26.07/helm/. Helm chart page: https://github.com/NetApp/ontap-mcp/blob/main/charts/ontap-mcp/Chart.yaml.

- :tophat: The ONTAP-MCP server exposes mutating tools in two naming conventions, controlled by the `--tool-mode` flag. **Note:** `--tool-mode` with value `multiplex` reduces the MCP tool count. Details: https://netapp.github.io/ontap-mcp/26.07/tools/#tool-mode

- The ONTAP-MCP server includes `--json-response` CLI flag. It's required when MCP server deploying behind proxies or gateways that do not relay SSE/chunked responses. Thanks @zlucas-netapp for contributing.

- :gem: **Tools enhancements** — couple of existing tools include new fields. Thanks @dbtinsley for contributing.
    - **LUN: create**: new space.guarantee.requested field in create, which enables thick provisioning at the LUN level.
    - **QoS policy: create, update, modify**: new expected_iops_allocation, peak_iops_allocation and block_size fields on adaptive QoS policies.
    - **Volume: create, update, modify**: new guarantee, snapshot policy, snapshot reserve and efficiency fields in volume.
    - **IGroup: create**: new initiators field in IGroup

- Join [Discord and GitHub discussions](https://github.com/NetApp/ontap-mcp/blob/main/SUPPORT.md) to participate in the conversation, ask questions, and share your feedback.

- :closed_book: Documentation is available at https://netapp.github.io/ontap-mcp/26.07/ and full list of tools is available at https://netapp.github.io/ontap-mcp/26.07/tools/.

- :closed_book: Added documentation regarding user permission requirements for the cluster. Thanks @Cool-hand-Kyle for raising.

Examples showing how you can manage ONTAP: https://netapp.github.io/ontap-mcp/26.07/examples/

## Thanks to all the awesome contributors

:metal: Thanks to all the people who've opened issues, asked questions on Discord, and contributed code for this release:

@werenzo, @zlucas-netapp, @jbnetapp, @ReBaunana, @dbtinsley, @tasosnetapp, @calvinwonghk, @Cool-hand-Kyle, @rahulguptajss, @cgrinds, @Hardikl

:seedling: This release includes 17 feature commits, 5 fixes, 3 documentation updates, 14 CI changes, 8 refactoring commits, and 11 miscellaneous merge commits.

<details>

<summary>Expand for full list of changes</summary>

### :rocket: Features
- Add initiators field to create_igroup for atomic create-with-membership (#178) ([1c0cb03](https://github.com/NetApp/ontap-mcp/commit/1c0cb03982fad8993ca6101c6d42c7c4b11f45f9))
- Add volume guarantee, snapshot policy, snapshot reserve, and efficiency fields (#179) ([f2d87ee](https://github.com/NetApp/ontap-mcp/commit/f2d87ee7b6f57df3fc241e7f9fd9ab1ad4e77a5c))
- Add expected_iops_allocation, peak_iops_allocation, and block_size to adaptive QoS (#180) ([511ebf4](https://github.com/NetApp/ontap-mcp/commit/511ebf46c2286b6e6192f4c6f894164717a392d9))
- Add space_guarantee_requested field to create_lun (#181) ([3bc75e3](https://github.com/NetApp/ontap-mcp/commit/3bc75e3f39f82d69479c6f27a254afd03bf8b692))
- Add optional HTTP listener to ListenerSet for ACME HTTP-01 (#187) ([cf96463](https://github.com/NetApp/ontap-mcp/commit/cf96463b38adca06d9867a5c1c0c547be9f0dc5f))
- Publish helm charts (#184) ([7283728](https://github.com/NetApp/ontap-mcp/commit/7283728287a52d99b9de993818897c734e4a47db))
- Merge tools with tool_mode flag (#157) ([404da86](https://github.com/NetApp/ontap-mcp/commit/404da86e52e69443b68c787a554f7cf021b62181))
- Add Helm chart for Kubernetes deployment (#168) ([ee93184](https://github.com/NetApp/ontap-mcp/commit/ee931846d09d70851234b60d45878e5197f18c59))
- Add version info to build (#162) ([5c56567](https://github.com/NetApp/ontap-mcp/commit/5c565676432064a99ef8d13025a2323cec2f6cab))
- Add top level tls section for https support (#158) ([e93aee7](https://github.com/NetApp/ontap-mcp/commit/e93aee776e86d44067ca855c05445bd2c810a863))
- Add oauth support (#144) ([f44c2d5](https://github.com/NetApp/ontap-mcp/commit/f44c2d5e706c2fd0e2d7fef770fa9e51887494e1))
- Config defaults work with poller key/values (#152) ([c126db2](https://github.com/NetApp/ontap-mcp/commit/c126db21f08abe8acc14c1ed0425e178dc5cd41f))
- Handled copilot comments ([1c4a718](https://github.com/NetApp/ontap-mcp/commit/1c4a718cdd1ff6732a722a91155943b8b5db657b))
- Handled dns test with skip validation ([6c67cdf](https://github.com/NetApp/ontap-mcp/commit/6c67cdf492e4323b0d663d8c1504d28cf1be076d))
- Add NFS service, CIFS service, and DNS tools ([147cf67](https://github.com/NetApp/ontap-mcp/commit/147cf67d45baed7712153b208f2550e9847ad99e))
- Implement case-insensitive cluster name handling (#136) ([64702eb](https://github.com/NetApp/ontap-mcp/commit/64702eb5e93bec1ed3b4307df9b61ecb70bf130f))
- Add --json-response CLI flag ([aad8d14](https://github.com/NetApp/ontap-mcp/commit/aad8d148088e561d557cd53baa525be1efd14e6d))


### :bug: Bug Fixes
- Handled lint issue (#191) ([47334df](https://github.com/NetApp/ontap-mcp/commit/47334dfae49a836feb2217fdf419cf7892c38a41))
- Add handleJob to DNS delete, add NFS update verification ([7c2d562](https://github.com/NetApp/ontap-mcp/commit/7c2d56262955bb85917e2e8014a80c131dc49053))
- Remove start-ontap-mcp.bat from tracking, validate AD credential pair ([eebdf1b](https://github.com/NetApp/ontap-mcp/commit/eebdf1b461f55e5163a010d87a51c7ad104d2c34))
- Format rest/svm.go, add job link support to DNS create ([a517da1](https://github.com/NetApp/ontap-mcp/commit/a517da1ea712dd6c8cb705dda6b03ed5da6d863f))
- Address PR review comments ([708f59b](https://github.com/NetApp/ontap-mcp/commit/708f59b5e0c4750604802922ebb09a7a4cd13c3c))


### :closed_book: Documentation
- Add Helm docs (#183) ([2ed8357](https://github.com/NetApp/ontap-mcp/commit/2ed8357ca8e3f867fa6f2204590af27cb1a40210))
- User permission for clusters in ontap mcp (#165) ([652cde8](https://github.com/NetApp/ontap-mcp/commit/652cde8bd7e9dd94b485901975e91aa1b82f4b63))
- Add version doc (#163) ([d01ed2e](https://github.com/NetApp/ontap-mcp/commit/d01ed2e2b8849bcc96e3f18c22411235b9c7e5af))


### :hammer: CI / Testing
- Remove untagged charts (#185) ([6b2c741](https://github.com/NetApp/ontap-mcp/commit/6b2c741457be554fefcfd10e574f35012a07575a))
- Dbtinsley has signed the CCLA ([4794a10](https://github.com/NetApp/ontap-mcp/commit/4794a1001b7d538688b29d9eabcb84d5da76c48b))
- ReBaunana has signed the CCLA ([deab6e9](https://github.com/NetApp/ontap-mcp/commit/deab6e9b50038fe81f00e706039d8347cf6a8d2b))
- Bump go (#166) ([dc55f43](https://github.com/NetApp/ontap-mcp/commit/dc55f431c421c7d5dc85471f2a4e7c75d4892ed9))
- Merge branch 'tasosnetapp-feature/nfs-cifs-service-tools' ([8037444](https://github.com/NetApp/ontap-mcp/commit/8037444adeb4ab85e89e6f06eaeb404efbb15bad))
- Bump go (#147) ([febd116](https://github.com/NetApp/ontap-mcp/commit/febd116974fc966bf974325d0e2b577d6d0b65bd))
- Tasosnetapp can contribute code (#146) ([2d04a2e](https://github.com/NetApp/ontap-mcp/commit/2d04a2ee406ad4d3ab6f35d9ce29a89a9b4ba0ec))
- Add DNS field validation in integration test ([942fbda](https://github.com/NetApp/ontap-mcp/commit/942fbdafdac0b9b7d49c54816dc2f726c6798090))
- Bump go (#147) ([bb3496f](https://github.com/NetApp/ontap-mcp/commit/bb3496f776190f95b8d361bcbf245bff3c3b0602))
- Tasosnetapp can contribute code (#146) ([fb25395](https://github.com/NetApp/ontap-mcp/commit/fb25395c0b2d7480137986f6a002c4889888c295))
- Bump modelcontextprotocol/go-sdk (#134) ([b55afbb](https://github.com/NetApp/ontap-mcp/commit/b55afbbb35d164617986002a5c287ba3d0ea2005))
- Bump go (#133) ([36f1106](https://github.com/NetApp/ontap-mcp/commit/36f11069b0fbf1e4f6082addf7f70d22808cdd49))
- Bump go-sdk to v1.6.0-pre.1 to address https://github.com/NetApp/ontap-mcp/discussions/126 ([06a3289](https://github.com/NetApp/ontap-mcp/commit/06a32898a28f8f2d292db3fdb9594a445faf6ccd))
- Zlucas-netapp can contribute code ([acee956](https://github.com/NetApp/ontap-mcp/commit/acee95647d3071ec11258ee18954c3624d52cc1d))


### Refactoring
- Handled copilot comments ([ae744f6](https://github.com/NetApp/ontap-mcp/commit/ae744f656fecda2b999b0ca1aae371644c8fe479))
- Handled review comments ([a447690](https://github.com/NetApp/ontap-mcp/commit/a447690bf2078c87e5dd10faa4c18c870e87de68))
- Handled copilot comments ([6be7e45](https://github.com/NetApp/ontap-mcp/commit/6be7e45385e05964bff608281d0191138a2ffc2a))
- Handled snapmirror tool path ([7286abf](https://github.com/NetApp/ontap-mcp/commit/7286abfb1b237463b2ebe3e68f73afff1abdbe5e))
- Update description of update tools (#155) ([54dd997](https://github.com/NetApp/ontap-mcp/commit/54dd997b149f92f3ee0deb06d9c57972393afb58))
- Adapt tests to use InsecureTLS() ([fd2592c](https://github.com/NetApp/ontap-mcp/commit/fd2592c3e87a9cbaa132120a8e1d8b4a79aceafe))
- Extract getSVMUUID helper, fix integration tests ([0f54214](https://github.com/NetApp/ontap-mcp/commit/0f54214a5def54043fff2e87b3b508a80bff7c5a))
- Disable nvme tools in ci (#139) ([0e4cc7a](https://github.com/NetApp/ontap-mcp/commit/0e4cc7a4e9de3672e8d31126049307b9d8b6206d))


### Miscellaneous
- Update all dependencies (#192) ([31a5236](https://github.com/NetApp/ontap-mcp/commit/31a5236306d7b8f9036ab5ce4ab2876470f898c7))
- Update all dependencies ([37852c8](https://github.com/NetApp/ontap-mcp/commit/37852c8013e7c8b3c8739d7d960a828ca569c010))
- Update actions/setup-go action to v6.5.0 (#164) ([0aa18d0](https://github.com/NetApp/ontap-mcp/commit/0aa18d078f9c36fb4dcbeafefb166d0cc44972a1))
- Update actions/checkout action to v7 ([1e08024](https://github.com/NetApp/ontap-mcp/commit/1e080240eee7de6d6b75aac44667da79bbb92757))
- Update all dependencies to v6.1.2 ([57bb911](https://github.com/NetApp/ontap-mcp/commit/57bb911ced40e802f022b08defc51e20a4b40e50))
- Update all dependencies ([a9d18b7](https://github.com/NetApp/ontap-mcp/commit/a9d18b7059388f8456e3fde2f8a5227873f0939c))
- Remove convenience scripts from repo, add integration tests ([7505745](https://github.com/NetApp/ontap-mcp/commit/750574593a3b9652b874dcfea5d107a57d884f28))
- Update all dependencies (#143) ([bc7ed9b](https://github.com/NetApp/ontap-mcp/commit/bc7ed9b2d624b9ecd99adbfe3c70e7d580a3eba6))
- Update all dependencies (#142) ([99332e8](https://github.com/NetApp/ontap-mcp/commit/99332e88883c0ad7a841c4809860fc93fdc16dd8))
- Update github/codeql-action action to v4.35.4 (#140) ([e0ebb84](https://github.com/NetApp/ontap-mcp/commit/e0ebb84bd2cf13d0a96b396e6dc4bc8757f0c53a))
- Update all dependencies (#135) ([b590d70](https://github.com/NetApp/ontap-mcp/commit/b590d702fea2aae952eac0e1d1569d0711e82dad))

</details>


---

## 26.04.0 / 2026-04-30 Release

The ONTAP-MCP team is happy to announce that we've released the 26.04.0 version of ONTAP-MCP. 🤘

- :medal_sports: The ONTAP-MCP server gives MCP clients like GitHub Copilot, Claude Desktop, and other large language models (LLMs) access to your NetApp ONTAP storage systems. It supports multi-cluster management and full protocol coverage across NAS, SAN block, and NVMe-oF. 

- :tophat: Each MCP tool is annotated with behavior hints (create/read/update/delete annotations) so clients can reason about safety.  
 
- :gem: This version also includes a Swagger-driven catalog that lets AI clients explore any ONTAP GET REST endpoint we haven't already wrapped.

- Join [Discord and GitHub discussions](https://github.com/NetApp/ontap-mcp/blob/main/SUPPORT.md) to participate in the conversation, ask questions, and share your feedback.

- :closed_book: Documentation is available at https://netapp.github.io/ontap-mcp/26.04/ and full list of tools is available at https://netapp.github.io/ontap-mcp/26.04/tools/.

Examples showing how you can manage ONTAP from Visual Studio Code: https://netapp.github.io/ontap-mcp/26.04/examples/

## Thanks to all the awesome contributors

:metal: Thanks to all the people who've opened issues, asked questions on Discord, and contributed code for this release:

@jbnetapp, @zlucas-netapp, @dmaryuma-ops, @NANAMINER, @ebarron, @dbenadiba, @calvinwonghk, @Antvirf, @leejshades

:seedling: This period includes 19 feature commits, 8 fixes, 3 documentation updates, 22 CI changes, and 3 refactoring pull requests.

<details>

<summary>Expand for full list of changes</summary>

### :rocket: Features
- Add stateless configuration ([5d90ddf](https://github.com/NetApp/ontap-mcp/commit/5d90ddf12156d05043777a46e1231bc4439befbc))
- Add SnapMirror tools ([e0b2a10](https://github.com/NetApp/ontap-mcp/commit/e0b2a10fe16d3e54f7ac5629e5a85f4143f03982))
- Add snapshot CRUD tools ([9f64548](https://github.com/NetApp/ontap-mcp/commit/9f6454848e47def6f22d006a03b29f058b134d86))
- Implement igroup CRUD tools ([fda3bab](https://github.com/NetApp/ontap-mcp/commit/fda3bab8b53ae2f1940bbf529904955bcbf95967))
- Add snapshot policy schedule create, update, and delete support ([df9e89b](https://github.com/NetApp/ontap-mcp/commit/df9e89b792c98bf5954080ac3c2d31aeb4ad94f3))
- Add FCP service and FC interface tools ([633d944](https://github.com/NetApp/ontap-mcp/commit/633d944a62c270ed030168ce9413060b212fb90a))
- Minor feature follow-up ([5b68e47](https://github.com/NetApp/ontap-mcp/commit/5b68e4718608c1378b27d207468cdd2d2ab92871))
- Handle Copilot review comments in feature work ([9102a34](https://github.com/NetApp/ontap-mcp/commit/9102a34b4c7c7537f5379cbca163427419085b9e))
- Add SVM update support for rename, state, and comment ([ac6046e](https://github.com/NetApp/ontap-mcp/commit/ac6046e2ae9ed7f08042b3adfcc67711692689cd))
- Add LUN CRUD tools with tests ([b4e2bce](https://github.com/NetApp/ontap-mcp/commit/b4e2bce776435218eb00d3b403e6ef94c63593ee))
- Add NVMe subsystem host tools ([3d63a61](https://github.com/NetApp/ontap-mcp/commit/3d63a611b7478175efb09521cb0e294d4acbd632))
- Add SVM tools and consume them in iSCSI tests ([0c88e3a](https://github.com/NetApp/ontap-mcp/commit/0c88e3ad3a95085960c9b8352809542e8895554b))
- Add iSCSI service tool ([f3c7f07](https://github.com/NetApp/ontap-mcp/commit/f3c7f07a23b40cc95c2083b75c711d62ab234287))
- Add NVMe service tool ([969f7d7](https://github.com/NetApp/ontap-mcp/commit/969f7d7fca1caa112721e5fd09fd1013679716c1))
- Add snapshot listing support ([a5d9a54](https://github.com/NetApp/ontap-mcp/commit/a5d9a547c8dcaf95d158977d116cd17cb811db90))
- Return cluster-scoped QoS policies in responses ([8db1f9a](https://github.com/NetApp/ontap-mcp/commit/8db1f9a3a74bb344694c500ac7570cf777802834))
- Add qtree tools ([6b28625](https://github.com/NetApp/ontap-mcp/commit/6b28625e289d1c97afda81b71a8ea1eb74914675))
- Add QoS apply and modify support for volumes ([54a66ec](https://github.com/NetApp/ontap-mcp/commit/54a66ec9b012ff5a510390ccc82b14b9cb457316))
- Add Swagger-driven guidance for GET calls ([013e518](https://github.com/NetApp/ontap-mcp/commit/013e518b9fc4c4fe3fc459706473c8318edca323))

### :bug: Bug Fixes
- Handle panic in tools ([0835bd3](https://github.com/NetApp/ontap-mcp/commit/0835bd3fec229bde30623cf6098627cd85b8f795))
- Use pointers to remove policy limit ([83487ec](https://github.com/NetApp/ontap-mcp/commit/83487ec7d8c90e807e2f9945b268dc44e835796f))
- Flush the inspect-traffic bytes buffer correctly ([1d4de58](https://github.com/NetApp/ontap-mcp/commit/1d4de58fdba64ddef2c537d83ecd1494e4cf6206))
- Work around missing empty properties when tool handler input is an array ([42b2ce6](https://github.com/NetApp/ontap-mcp/commit/42b2ce63e6409754fa699fe4f98da4ddb39e7f5e))
- Remove private fields from the catalog ([7d922d3](https://github.com/NetApp/ontap-mcp/commit/7d922d346235de40225996cc67ad4ab1a82fc1ea))
- Limit token usage to the dedicated environment variable ([79cc1d5](https://github.com/NetApp/ontap-mcp/commit/79cc1d5daf659dde5aaec74faf47d2b79f9f47a9))
- Fix empty properties in list-clusters responses ([06ceb37](https://github.com/NetApp/ontap-mcp/commit/06ceb37475875ab0fa3e3ca1f0460449b29582db))
- Re-apply the policy-limit pointer fix after rollback ([89836f2](https://github.com/NetApp/ontap-mcp/commit/89836f2ce8c6828c10231eecb6e240c2b447f7ac))

### :closed_book: Documentation
- Remove early access messaging ([24db812](https://github.com/NetApp/ontap-mcp/commit/24db8124412a4dfa8e0f9d9e0c4e70486c7b6706))
- Add tool documentation ([7a47562](https://github.com/NetApp/ontap-mcp/commit/7a4756263d15ed8d54be36075e6e7701ef00acdf))
- Remove the redundant CIFS section ([f6f3c2a](https://github.com/NetApp/ontap-mcp/commit/f6f3c2aed121cad16903e44fb15cbd2e6e486313))

### :hammer: CI / Testing
- Add `-e` handling for mkdir errors ([0af8cef](https://github.com/NetApp/ontap-mcp/commit/0af8cefdf2cfdc74f72ed448ddaa465befdf0c31))
- Add a Jenkins task for release automation ([57fefcf](https://github.com/NetApp/ontap-mcp/commit/57fefcf49c32f95c1e365043c932d44b833e5840))
- Add session handling to the agent workflow ([9b00b1d](https://github.com/NetApp/ontap-mcp/commit/9b00b1d73af508f143eb1e0959d59eb890ff119f))
- Add CLA bot and Renovate bot automation ([eaef368](https://github.com/NetApp/ontap-mcp/commit/eaef368147f5f103cd6366b7e7978c17e80754e5))
- Add support for running integration tests in parallel ([6be9006](https://github.com/NetApp/ontap-mcp/commit/6be9006d268e0942bc95aeb8ecbd89fe83dd626c))
- Add Docker cleanup steps ([8df8d73](https://github.com/NetApp/ontap-mcp/commit/8df8d734e3c8bbd4653d69529211a1870a93d604))
- Bump CI dependencies ([7ea47b5](https://github.com/NetApp/ontap-mcp/commit/7ea47b5174cbdca6426271368c3a9dd273663aa9))
- Improve the health check ([c31122a](https://github.com/NetApp/ontap-mcp/commit/c31122a3e1bc68026d8aefd67d6d8f1eb939847d))
- Add junction path support in volume tooling ([82f9ac4](https://github.com/NetApp/ontap-mcp/commit/82f9ac41814342434072025dcbd3c05461bcedf4))
- Handle CI review comments ([7c2c5ff](https://github.com/NetApp/ontap-mcp/commit/7c2c5ff5a6a032d309b4011ffef9030a0ad0c9cf))
- Lint integration files ([b89910c](https://github.com/NetApp/ontap-mcp/commit/b89910cba14e4b515aee68f3aa4860387a54212f))
- Verify actual object operations in ONTAP test cases and add MCP health coverage ([4c09203](https://github.com/NetApp/ontap-mcp/commit/4c09203850c253b290fb9cc8b14192e9b65a7c5b))
- Convert tests to a tabular layout ([2678f2e](https://github.com/NetApp/ontap-mcp/commit/2678f2e2308a1a8226e501f0a1a1ab472a5ae063))
- Minor CI follow-up ([a17c74a](https://github.com/NetApp/ontap-mcp/commit/a17c74a2727c083df5072e6610e5f2299f14e64e))
- Update smoke test execution for pull requests ([dd187e3](https://github.com/NetApp/ontap-mcp/commit/dd187e3f1de15a406bdff20fc9a4713c0b7a4dda))
- Update the rule-update tool workflow ([c6c03b1](https://github.com/NetApp/ontap-mcp/commit/c6c03b13b7df94879fcd59ac628436bb8045aa88))
- Rename a CI variable for clarity ([2ee2129](https://github.com/NetApp/ontap-mcp/commit/2ee212920e426a21cc09afae5c1b7db8caf3215e))
- Update file permissions in CI ([11a7f76](https://github.com/NetApp/ontap-mcp/commit/11a7f768d599837bb04599c496bb3d703e187b28))
- Minor CI follow-up ([1b30d8b](https://github.com/NetApp/ontap-mcp/commit/1b30d8bd04f119948e3f5cb7c1e4dba7d85db544))
- Minor CI follow-up ([dc21bc7](https://github.com/NetApp/ontap-mcp/commit/dc21bc7d52cdd48fb75d819f920867252354edbb))
- Minor CI follow-up ([d5fff1e](https://github.com/NetApp/ontap-mcp/commit/d5fff1eeb995898a7e729a28a6bc8666fb387a56))
- Add a tool test case for the LLM proxy path ([d50d68d](https://github.com/NetApp/ontap-mcp/commit/d50d68ddd92b974bf469862add5f072c8210216f))

### Refactoring
- Downgrade `ontap_get` retry errors to debug logging ([37109a9](https://github.com/NetApp/ontap-mcp/commit/37109a970188e2b40f655d8990bb2bb790e116dc))
- Move volume tools into the volume file ([c473e1e](https://github.com/NetApp/ontap-mcp/commit/c473e1e9139bad193b0e7da7bfb596f0fad1f7e8))
- Move tests to their relevant files ([0b71e7a](https://github.com/NetApp/ontap-mcp/commit/0b71e7ab9c7f51ef64751d767c6ad4ca1931affe))

### Miscellaneous
- Update all dependencies ([a2e1e23](https://github.com/NetApp/ontap-mcp/commit/a2e1e23cccefde9d704ea4e8f2b1219870df96e4))
- Update all dependencies ([554c049](https://github.com/NetApp/ontap-mcp/commit/554c0491bdd7e3f721ff7d2eae4784f70324d93b))
- Update all dependencies ([64451e2](https://github.com/NetApp/ontap-mcp/commit/64451e25b397b26e92dece88add59610612456b5))
- Migrate the Renovate configuration ([c14ebe6](https://github.com/NetApp/ontap-mcp/commit/c14ebe6683786186484665194c3baefa3bcd8499))
- Bump Go ([36d430b](https://github.com/NetApp/ontap-mcp/commit/36d430b25f298d90bc51937a39ce1f8e85b36927))
- Update the environment file ([7606c94](https://github.com/NetApp/ontap-mcp/commit/7606c9446753712a13f978fbc68eae8d1b784881))
- Update Go ([d252d10](https://github.com/NetApp/ontap-mcp/commit/d252d10dfa504b06db38262b0ee8791ee0d0dd71))
</details>
