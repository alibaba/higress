# Zodiac Analysis

The APP Code required for API authentication can be applied for on the Alibaba Cloud API Marketplace: https://market.aliyun.com/apimarket/detail/cmapi011529

# Zodiac Analysis Server Function Introduction Document

This document aims to provide a detailed explanation of the Zodiac Analysis server and its tools, helping users better understand their functions and application scenarios.

## Function Overview

The Zodiac Analysis server is a service platform specifically designed for querying zodiac horoscopes. It can return corresponding horoscope information based on user request parameters, such as a specific zodiac sign and the desired time range (e.g., day, week, month, or year). This service is particularly useful for those who wish to obtain personalized horoscope predictions through technical means, such as astrology enthusiasts, app developers, or website operators looking to enhance user experience.

## Tool Introduction

### Zodiac Horoscope Query

- **Purpose**: This tool allows users to query the horoscope for a specified zodiac sign over different time periods.
- **Use Cases**:
  - When individual users want to know about their own or others' recent horoscope trends;
  - Application developers can integrate this service to increase the interactivity and appeal of their products;
  - Website administrators can embed this feature to boost visitor engagement.
- **Request Parameters**:
  - `needMonth` (whether data for the current month is needed, 1 for yes)
  - `needTomorrow` (whether data for tomorrow is needed, 1 for yes)
  - `needWeek` (whether data for the current week is needed, 1 for yes)
  - `needYear` (whether data for the current year is needed, 1 for yes)
  - `star` (one of the twelve zodiac signs, required)

> Note: All parameters are located in the URL's query section.

- **API Call Example**:
  - Request template URL: `https://luck141219.market.alicloudapi.com/star`
  - Method: GET
  - HTTP headers that need to be set include Authorization (the value is determined by the appCode in the configuration) and X-Ca-Nonce (dynamically generated).

- **Response Structure Overview**:
  - Response content type: application/json
  - Main fields:
    - `showapi_res_body.day_notice`: Today's reminder
    - `showapi_res_body.day.general_txt`: General horoscope review
    - `showapi_res_body.day.love_star`: Love index (out of 5 points)
    - For more details, please refer to the complete response structure description provided above.

From the above introduction, it can be seen that the Zodiac Analysis server not only provides flexible zodiac horoscope query functionality but also supports various customization options, meeting the diverse needs of both general users and professional developers.