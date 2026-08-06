# Document Conversion

The APP Code required for API authentication can be applied for on the Alibaba Cloud API Marketplace: https://market.aliyun.com/apimarket/detail/cmapi00067671

# MCP Server Configuration Documentation

## Overview
This MCP server is primarily used to provide file format conversion services, supporting the conversion of PDF files into Word, PPT, or Excel formats, as well as converting common office document formats such as Word, Excel, PPT, and txt into PDF. Additionally, it provides a feature to query the results of file conversions, allowing users to track the status of their requests. With these tools, users can easily convert between different document types and add watermarks as needed to protect the content of documents.

## Tool Introduction

### PDF to Document
This tool allows users to convert PDF files into various Microsoft Office document formats, including but not limited to Word (.docx, .doc), PowerPoint (.pptx, .ppt), and Excel (.xlsx, .xls) files. It is ideal for situations where information needs to be extracted from a fixed-layout PDF and edited.
- **callBackUrl**: The callback URL for receiving notifications upon completion of the conversion.
- **fileUrl**: The URL link to the PDF file to be converted; there are certain limitations on file size and number of pages.
- **type**: Specifies the target output file format.

### Document to PDF
With this feature, users can create PDF versions from Word, Excel, PPT, and even plain text files. This is very useful for scenarios where cross-platform compatibility or enhanced security is desired. In addition to basic conversion capabilities, it also supports adding custom text or image watermarks to the generated PDF.
- **callBackUrl**: The notification address for receiving updates on the conversion status.
- **fileUrl**: The source file link that needs to be converted to PDF; different types of files have different maximum size limits.
- **watermarkColor**, **watermarkFontName**, **watermarkFontSize**, **watermarkImage**, **watermarkLocation**, **watermarkRotation**, **watermarkText**, **watermarkTransparency**: These parameters collectively define how the watermark effect will be displayed in the final PDF document.

### Document Conversion Result Query
After a conversion task is submitted, it may take some time to complete. This API allows users to check the status of a specific conversion request, thereby understanding whether the conversion was successful and obtaining any download links for the generated files.
- **convertTaskId**: The task identifier returned from a previously initiated conversion request, used to track the processing progress.

All of the above tools are invoked via POST requests and require setting the appropriate content type header (Content-Type: application/x-www-form-urlencoded) and authorization information (Authorization: APPCODE [appCode]) to access the relevant service endpoints on the Alibaba Cloud Marketplace. Each response includes information about the success or failure of the operation, and if applicable, provides a direct link to the converted file.