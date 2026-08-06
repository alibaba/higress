# Invoice Verification

The APP Code required for API authentication can be applied for on the Alibaba Cloud API Marketplace: https://market.aliyun.com/apimarket/detail/cmapi00050226

# MCP Server Function Overview Document

## Function Overview
This MCP server is primarily responsible for handling various verification and download tasks related to invoices. Through a series of tools, it can achieve functions such as blockchain invoice verification, invoice downloading, and invoice checking. This server is suitable for enterprises and organizations that need to verify, download, or further process invoice information. The configuration file defines multiple tools, each with specific functions and application scenarios.

## Tool Introduction

### 1. Blockchain Invoice Verification
**Purpose:**
Used to verify the authenticity of invoices issued based on blockchain technology.
  
**Use Case:**
When an enterprise receives an electronic invoice based on blockchain technology, this tool can be used to confirm the validity and accuracy of the invoice. It supports verification based on region, invoice code, number, etc.

### 2. Invoice Download v2
**Purpose:**
Provides a service to obtain and download specified invoice format files (such as PDF, OFD) from the cloud.
  
**Use Case:**
After completing the invoice verification, users can use this tool to quickly download the corresponding electronic invoice copy for archiving or subsequent processing. Input parameters include the invoice number, total amount including tax, and the invoicing date.

### 3. Invoice Checking V2
**Purpose:**
Performs detailed information queries and authenticity checks for different types of invoices (such as VAT special/general invoices, etc.).
  
**Use Case:**
Suitable for finance departments to review various types of invoices submitted by employees before reimbursement, ensuring all data is accurate. Detailed invoice content, such as the name of the buyer, seller, and amount, can be queried.

### 4. Invoice Validation
**Purpose:**
Simply determines whether an invoice is legal and valid based on the provided basic invoice information (such as invoice code, number, etc.).
  
**Use Case:**
Suitable for preliminary screening of a large number of invoices for authenticity, especially for applications that only require basic validation without in-depth detail analysis.

### 5. Fiscal Receipt Validation
**Purpose:**
Specifically used to verify the authenticity and integrity of various receipts issued by the finance department.
  
**Use Case:**
Government agencies or related institutions can use this tool to ensure that the fiscal receipts used in transactions involving public funds are genuine and official documents.

### 6. Vehicle Toll Invoice Verification_Jiangsu
**Purpose:**
Provides online verification services specifically for vehicle toll invoices within Jiangsu Province.
  
**Use Case:**
Logistics companies or businesses that frequently need to transport goods across cities can use this tool to verify the accuracy of the toll invoices they receive when settling transportation fees.

### 7. General Electronic Invoice Verification
**Purpose:**
A more broadly applicable solution for electronic invoice verification, not limited to specific types or regions of invoices.
  
**Use Case:**
Any enterprise that needs to conduct comprehensive and detailed checks on electronic invoices can adopt this service, especially those with a wide business scope and diverse customer base.