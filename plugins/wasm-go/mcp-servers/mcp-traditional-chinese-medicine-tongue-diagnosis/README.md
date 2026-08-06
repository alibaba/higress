# Traditional Chinese Medicine Tongue Diagnosis

The APP Code required for API authentication can be applied for on the Alibaba Cloud API Marketplace: https://market.aliyun.com/apimarket/detail/cmapi00069588

# MCP Server Configuration Document for Traditional Chinese Medicine Tongue Diagnosis

## Overview of Features

### 1. Project Summary
- **Name**: `traditional-chinese-medicine-tongue-diagnosis`
- **Main Function**: Provides AI-based traditional Chinese medicine tongue image recognition, constitution detection, and health advice services.
- **Application Scenarios**: Suitable for user groups who wish to obtain personal health status assessments through non-invasive means. It is particularly suitable for individuals who are concerned about their health but are unwilling or unable to go to a hospital for a comprehensive check-up.

### 2. Key Features
- Supports automatic analysis of user's constitutional features based on uploaded tongue photos.
- Provides detailed interpretation of diagnostic results, including but not limited to acupoint treatment recommendations, lifestyle adjustment guidance, etc.
- Offers dietary therapy recommendations tailored to different constitutions.
- User-friendly interface design, easy to integrate into existing applications.

## Tool Introduction

In this MCP server configuration, a core tool named "AI Tongue Diagnosis - Tongue Image Recognition - Constitution Detection - Health Report" is defined, which handles all requests related to traditional Chinese medicine tongue diagnosis.

### AI Tongue Diagnosis - Tongue Image Recognition - Constitution Detection - Health Report
- **Description**: This tool can receive personal information and the URL of a tongue image provided by the user, and after AI algorithm analysis, it returns a detailed health report.
- **Input Parameters**:
  - `customerAge` (Customer Age): Required, integer type, used to more accurately assess health status.
  - `customerSex` (Customer Gender): Required, string type, helps the system understand individual differences.
  - `frontRear` (Front/Back Position Identifier): Required, string type, indicates the shooting angle.
  - `situation` (Current Situation Identifier): Required, integer type, describes factors that may affect the result under special conditions.
  - `tonguePic` (URL of the Tongue Picture): Required, string type, should point to a clear and visible front-facing picture of the tongue.
- **API Call Information**:
  - **Request Method**: POST
  - **Target URL**: https://aizong.market.alicloudapi.com/symptomDiagnose/cloudResult
  - **Header Information**:
    - `Content-Type`: application/json
    - `Authorization`: Use the preset application code as the authentication token
    - `X-Ca-Nonce`: A uniquely generated identifier to ensure the security of each request

- **Response Structure**:
  - Contains an operation status code (`code`) and specific result data (`data`).
  - The result data lists in detail various characteristic analyses of the user's constitution, related acupoint information, recommended food therapies, etc.
  - Each item is accompanied by clear Chinese explanations, making it easy to understand and apply.

From the above introduction, it can be seen that this MCP server not only provides powerful data analysis capabilities but also places great emphasis on user experience, striving to make complex medical knowledge accessible and understandable.