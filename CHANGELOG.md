# Change Log
## [Repository](https://github.com/NetApp/ontap-mcp)

## 26.04.0 / 2026-04-30 Release

The ONTAP-MCP team is happy to announce that we've released the 26.04.0 version of ONTAP-MCP. 🤘

- :medal_sports: The ONTAP-MCP server gives MCP clients like GitHub Copilot, Claude Desktop, and other large language models (LLMs) access to your NetApp ONTAP storage systems. It supports multi-cluster management and full protocol coverage across NAS, SAN block, and NVMe-oF. 

- :tophat: Each MCP tool is annotated with behavior hints (create/read/update/delete annotations) so clients can reason about safety.  
 
- :gem: This version also includes a Swagger-driven catalog that lets AI clients explore any ONTAP GET REST endpoint we haven't already wrapped.

- Join [Discord and GitHub discussions](https://github.com/NetApp/ontap-mcp/blob/main/SUPPORT.md) to participate in the conversation, ask questions, and share your feedback.

- :closed_book: Documentation is available at https://netapp.github.io/ontap-mcp/26.04/ and full list of tools is available at https://netapp.github.io/ontap-mcp/26.04/tools/.

Examples showing how you can manage ONTAP from Visual Studio Code: https://netapp.github.io/ontap-mcp/26.04/examples/

## Announcements

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
