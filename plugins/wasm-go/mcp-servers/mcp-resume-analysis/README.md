# Resume Parsing

The APP Code required for API authentication can be applied for on the Alibaba Cloud API Marketplace: https://market.aliyun.com/apimarket/detail/cmapi00066399

# MCP Server Configuration Document

## Overview
The `resume-analysis` MCP server is primarily used for parsing resume files by calling the RuiShi engine's resume parsing interface to extract and structure key information from resumes. This service supports various types of resume files and can parse out detailed information such as the candidate's basic information, educational background, and work experience as needed. Additionally, the service provides an option to parse the profile picture in the resume.

## Tool Introduction
### Resume Parsing
- **Description**: Utilizes the API provided by the RuiShi engine to parse uploaded resumes.
- **Use Case**: Suitable for human resources departments or recruitment platforms to automatically process resumes submitted by candidates, quickly and accurately obtaining important data points such as personal information, education, and work experience.
- **Parameter Description**:
  - `file_content`: Required, the Base64-encoded content of the resume file.
  - `file_name`: The name of the file, used for identification and management of uploaded files.
  - `mode`: Parsing mode, which may specify different parsing strategies or versions.
  - `parse_avatar`: Whether to attempt to extract the profile picture from the resume, default is an integer type (0 means no parsing, 1 means parsing).

- **Request Example**:
  - URL: `https://qingsongai.market.alicloudapi.com/resume/parse`
  - Method: POST
  - Headers:
    - Content-Type: application/json
    - Authorization: APPCODE [Enter your APP Code here]
    - X-Ca-Nonce: A unique identifier generated automatically

- **Response Structure**:
  The response will include parsed resume information, including but not limited to basic information (`basic_info`), career list (`career_list`), educational background (`edu_list`), and contact information (`contact_info`). Each section is further divided into multiple fields, such as `name` and `age` under `basic_info`. Additionally, it includes a status code (`status.code`) and corresponding message (`status.message`) to indicate whether the request was successfully executed and the reason.

This tool greatly simplifies the process of manually entering resume information for HR personnel, improving work efficiency and ensuring data consistency and accuracy.