# Higress


## 📋 Overview of This Release

This release includes **11** updates, covering areas such as feature enhancements, bug fixes, performance optimizations, and more.

### Distribution of Updates

- **New Features**: 3
- **Bug Fixes**: 5
- **Refactoring and Optimization**: 1
- **Documentation Updates**: 2

---

## 📝 Full Changelog

### 🚀 New Features (Features)

- **Related PR**: [#2978](https://github.com/alibaba/higress/pull/2978) \
  **Contributor**: @rinfx \
  **Change Log**: In the key-auth plugin, regardless of whether authentication is successful, the consumer name will be recorded after it is determined. This is achieved by adding the X-Mse-Consumer field to the HTTP request header. \
  **Feature Value**: This feature allows the system to obtain and record the consumer's name earlier, which is very important for logging and subsequent processing, improving the traceability and transparency of the system.

- **Related PR**: [#2968](https://github.com/alibaba/higress/pull/2968) \
  **Contributor**: @2456868764 \
  **Change Log**: This PR introduces the core functionality of vector database mapping, including a field mapping system and index configuration management, supporting various index types. \
  **Feature Value**: By providing flexible field mapping and index configuration capabilities, users can more easily integrate with different database architectures, enhancing the system's compatibility and flexibility.

- **Related PR**: [#2943](https://github.com/alibaba/higress/pull/2943) \
  **Contributor**: @Guo-Chenxu \
  **Change Log**: Added a feature for customizing system prompts, allowing users to add personalized notes when generating release notes. This is implemented by modifying the GitHub Actions workflow file. \
  **Feature Value**: This feature allows users to include customized system prompts when generating release notes, enhancing the flexibility and richness of information in the release notes, thereby improving the user experience.

### 🐛 Bug Fixes (Bug Fixes)

- **Related PR**: [#2973](https://github.com/alibaba/higress/pull/2973) \
  **Contributor**: @CH3CHO \
  **Change Log**: This PR fixes an issue in Higress version 2.1.8 where the `mcp-session` filter did not support setting `match_rule_domain` to an empty string, using wildcards to match all domains and eliminate compatibility risks. \
  **Feature Value**: This resolves a compatibility issue caused by specific configurations, ensuring that users do not encounter errors due to empty string settings during upgrades or configuration, thus improving the stability and user experience of the system.

- **Related PR**: [#2952](https://github.com/alibaba/higress/pull/2952) \
  **Contributor**: @Erica177 \
  **Change Log**: Corrected the JSON tag for the Id field in the ToolSecurity struct from type to id, ensuring correct mapping during data serialization. \
  **Feature Value**: This fix addresses data inconsistency issues caused by incorrect field mapping, enhancing the stability and data accuracy of the system.

- **Related PR**: [#2948](https://github.com/alibaba/higress/pull/2948) \
  **Contributor**: @johnlanni \
  **Change Log**: Fixed the Azure service URL type detection logic to support custom full paths. Enhanced the handling of Azure OpenAI response APIs and improved edge case parsing in streaming events. \
  **Feature Value**: This ensures better compatibility with Azure OpenAI services, improves error handling and user experience, especially when using non-standard paths or streaming responses.

- **Related PR**: [#2942](https://github.com/alibaba/higress/pull/2942) \
  **Contributor**: @2456868764 \
  **Change Log**: Fixed the issue of LLM provider being empty and optimized documentation and prompt messages. Specifically, updated README.md for better explanations and adjusted the default LLM model. \
  **Feature Value**: By enhancing the robustness of LLM provider initialization and optimizing related documentation, this improves the stability and user experience of the system, making it clearer for users to understand system configuration and usage.

- **Related PR**: [#2941](https://github.com/alibaba/higress/pull/2941) \
  **Contributor**: @rinfx \
  **Change Log**: This PR fixes compatibility issues with old configurations, ensuring the system can correctly handle outdated configuration parameters, avoiding potential errors due to configuration changes. \
  **Feature Value**: By supporting older version configurations, this enhances the system's backward compatibility, reducing inconvenience to users during upgrades or configuration adjustments, and improving the user experience.

### ♻️ Refactoring and Optimization (Refactoring)

- **Related PR**: [#2945](https://github.com/alibaba/higress/pull/2945) \
  **Contributor**: @rinfx \
  **Change Log**: Optimized the logic for selecting pods based on the minimum number of requests globally, updated the Lua script code related to ai-load-balancer, reducing unnecessary checks and improving performance. \
  **Feature Value**: By improving the minimum request count algorithm in load balancing strategies, this enhances the system's response speed and resource allocation efficiency, allowing users to utilize cluster resources more efficiently.

### 📚 Documentation Updates (Documentation)

- **Related PR**: [#2965](https://github.com/alibaba/higress/pull/2965) \
  **Contributor**: @CH3CHO \
  **Change Log**: Updated the description of the azureServiceUrl field in the ai-proxy plugin README file to provide clearer and more accurate information. \
  **Feature Value**: By improving the description in the documentation, users can better understand how to configure the Azure OpenAI service URL, thus enhancing the user experience and configuration accuracy.

- **Related PR**: [#2940](https://github.com/alibaba/higress/pull/2940) \
  **Contributor**: @github-actions[bot] \
  **Change Log**: This PR adds English and Chinese release notes for version 2.1.8, detailing 30 updates in this version. \
  **Feature Value**: By providing detailed release notes, users can more easily understand the new features, bug fixes, and other information included in the new version, allowing them to make better use of the new features.

---

## 📊 Release Statistics

- 🚀 New Features: 3
- 🐛 Bug Fixes: 5
- ♻️ Refactoring and Optimization: 1
- 📚 Documentation Updates: 2

**Total**: 11 changes

Thank you to all contributors for their hard work! 🎉

# Higress Console


## 📋 Overview of This Release

This release includes **4** updates, covering multiple aspects such as feature enhancements, bug fixes, and performance optimizations.

### Update Distribution

- **New Features**: 1
- **Bug Fixes**: 2
- **Documentation Updates**: 1

### ⭐ Key Highlights

This release contains **1** significant update, which is recommended for special attention:

- **feat: Support using a known service in OpenAI LLM provider** ([#589](https://github.com/higress-group/higress-console/pull/589)): This feature allows users to use predefined services within the OpenAI LLM, thereby enhancing development efficiency and flexibility, and meeting the needs of a wider range of application scenarios.

For more details, please refer to the Important Feature Details section below.

---

## 🌟 Important Feature Details

Below are detailed explanations of key features and improvements in this release:

### 1. feat: Support using a known service in OpenAI LLM provider

**Related PR**: [#589](https://github.com/higress-group/higress-console/pull/589) | **Contributor**: [@CH3CHO](https://github.com/CH3CHO)

**Usage Background**

As more organizations and services adopt large language models (LLMs), access and management of these models have become increasingly important. Especially when integration with specific known services, such as an on-premises OpenAI API server or a custom API endpoint, is required. This feature addresses the need for direct support of custom OpenAI services within the Higress system, allowing users to more flexibly configure and use their services. The target user groups include, but are not limited to, developers, operations personnel, and enterprises requiring highly customized solutions.

**Feature Details**

The update primarily focuses on the `OpenaiLlmProviderHandler` class, introducing support for custom service sources. By adding new configuration options like `openaiCustomServiceName` and `openaiCustomServicePort`, users can now directly specify the details of their custom OpenAI service. Additionally, the code has been improved so that if a custom upstream service is specified, a service source will not be created for the default service. This design not only simplifies the configuration process but also enhances the system's scalability. Technically, this is achieved by overriding the `buildServiceSource` and `buildUpstreamService` methods, which include checks for user-defined settings.

**Usage Instructions**

To enable and configure this new feature, users first need to provide the necessary custom service information in their OpenAI LLM provider settings. This typically involves filling in fields such as the custom service name, host address, and port number. The general steps are: 1. Locate the relevant LLM provider settings section in the Higress console or corresponding configuration file; 2. Enter the appropriate custom service details as prompted; 3. Save the changes. A typical use case might be a company wishing to use its own internally hosted OpenAI interface instead of the publicly available one. It is important to ensure that the provided custom service address is accurate and network-accessible.

**Feature Value**

This feature greatly enhances the adaptability of the Higress platform to different environments, especially for scenarios requiring high levels of customization. It not only improves the user experience—making the configuration process more intuitive and simple—but also promotes the overall stability and security of the system, as it now allows for the direct use of trusted internal resources. In the long run, such enhancements help build a more robust ecosystem, encouraging more innovative application development.

---

## 📝 Full Changelog

### 🐛 Bug Fixes

- **Related PR**: [#591](https://github.com/higress-group/higress-console/pull/591) \
  **Contributor**: @CH3CHO \
  **Change Log**: Fixed the issue where required fields were not properly validated when enabling route rewriting, ensuring that both `host` and `newPath.path` must provide valid values when enabled. \
  **Feature Value**: This fix improves the accuracy and robustness of system configurations, preventing functional anomalies due to incomplete configurations and enhancing the user experience.

- **Related PR**: [#590](https://github.com/higress-group/higress-console/pull/590) \
  **Contributor**: @CH3CHO \
  **Change Log**: Fixed an error in the Route.customLabels processing logic, ensuring that built-in labels are correctly excluded during updates. \
  **Feature Value**: Resolved the conflict between custom labels and built-in labels when updating Routes, improving the stability and user experience of the system.

### 📚 Documentation

- **Related PR**: [#595](https://github.com/higress-group/higress-console/pull/595) \
  **Contributor**: @CH3CHO \
  **Change Log**: This PR updated the README.md file, removing non-project-level descriptions and adding code formatting guidelines. \
  **Feature Value**: By cleaning up irrelevant information and providing formatting suggestions, it helps developers better understand the project documentation, promoting consistency and readability in code contributions.

---

## 📊 Release Statistics

- 🚀 New Features: 1
- 🐛 Bug Fixes: 2
- 📚 Documentation Updates: 1

**Total**: 4 changes (including 1 significant update)

Thank you to all contributors for your hard work! 🎉

