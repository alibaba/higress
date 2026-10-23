# Moji Weather Query

The APP Code required for API authentication can be applied for on the Alibaba Cloud API Marketplace: https://market.aliyun.com/apimarket/detail/cmapi012364

# Function Overview Document

## Function Overview
The `weather-query` MCP server is a comprehensive weather query service designed to provide users with detailed weather information. This server supports various weather-related API tools, including but not limited to AQI forecasts, real-time weather data, and 15-day weather forecasts. Through these tools, users can obtain detailed air quality index (AQI), weather warnings, and life indices, helping them better plan their daily activities.

## Tool Overview

### 1. 5-Day AQI Forecast
- **Purpose**: Provides AQI data for the next 5 days.
- **Use Case**: Suitable for users or applications that need to understand the trend of air quality changes over the next few days, such as travel planning and outdoor activity scheduling.
- **Request Parameters**:
  - `lat` (Latitude): Required, used to specify the geographic coordinates of the location.
  - `lon` (Longitude): Required, used in conjunction with latitude.
  - `token`: Default parameter, must be filled in, used for authentication.

### 2. Current Weather Conditions
- **Purpose**: Provides real-time weather data for the current location, including temperature, humidity, wind speed, and other meteorological elements.
- **Use Case**: Suitable for applications that require immediate weather information, such as weather forecast apps and smart wearable devices.
- **Request Parameters**:
  - `lat` (Latitude): Required.
  - `lon` (Longitude): Required.
  - `token`: Required, used for authentication.

### 3. 15-Day Weather Forecast
- **Purpose**: Predicts the weather conditions for the next 15 days, including daily high and low temperatures and weather conditions.
- **Use Case**: Suitable for long-term travel planning, agricultural planting cycle management, and other fields.
- **Request Parameters**:
  - `lat` (Latitude): Required.
  - `lon` (Longitude): Required.
  - `token`: Required.

### 4. 24-Hour Weather Forecast
- **Purpose**: Provides hourly weather forecasts for the next 24 hours.
- **Use Case**: Very useful for services that require precise short-term weather information, such as flight scheduling and outdoor event organization.
- **Request Parameters**:
  - `lat` (Latitude): Required.
  - `lon` (Longitude): Required.
  - `token`: Required.

### 5. Weather Warnings
- **Purpose**: Issues extreme weather warning information for specific regions.
- **Use Case**: Disaster prevention systems, public safety notifications, etc.
- **Request Parameters**:
  - `lat` (Latitude): Required.
  - `lon` (Longitude): Required.
  - `token`: Required.

### 6. Life Index
- **Purpose**: Provides lifestyle guidelines based on current weather conditions, such as clothing suggestions and car washing indices.
- **Use Case**: Lifestyle applications, health management software, etc.
- **Request Parameters**:
  - `lat` (Latitude): Required.
  - `lon` (Longitude): Required.
  - `token`: Required.

### 7. Short-Term Forecast
- **Purpose**: Provides detailed weather forecasts for the next two hours.
- **Use Case**: Suitable for situations requiring short-term weather updates, such as temporary outdoor activities.
- **Request Parameters**:
  - `lat` (Latitude): Required.
  - `lon` (Longitude): Required.
  - `token`: Required.

### 8. Air Quality Index
- **Purpose**: Displays the concentration of major pollutants and the overall air quality index for the current area.
- **Use Case**: Environmental monitoring, health consultations, etc.
- **Request Parameters**:
  - `lat` (Latitude): Required.
  - `lon` (Longitude): Required.
  - `token`: Required.

### 9. Traffic Restriction Data
- **Purpose**: Provides vehicle traffic restriction information based on license plate number policies for certain cities.
- **Use Case**: Traffic management, travel assistant applications, etc.
- **Request Parameters**:
  - `lat` (Latitude): One of the important parameters.
  - `lon` (Longitude): One of the important parameters.
  - `token`: One of the important parameters.

Each of the above tools is invoked via the POST method, and all requests must include basic authentication information (i.e., `token`) to ensure the security and accuracy of the data.