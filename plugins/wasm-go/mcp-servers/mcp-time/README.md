# Time Service

This service provides accurate time information based on specified timezones. No API authentication is required.

# MCP Server Configuration Function Overview

## Function Overview

This service can provide the current time based on a specified timezone. It can also determine the user's location through their IP address, and then provide the accurate time based on the timezone of that location.

## Tool Introduction

### Get Current Time

- **Purpose**: This tool allows users to get the current time in a specific timezone or the system default timezone.
- **Use Cases**: Suitable for applications that need to display accurate local time to users in different regions, such as international websites, travel applications, or scheduling systems that operate across multiple time zones.
- **Request Parameters**:
  - `timeZone` (optional): IANA timezone name (e.g., 'America/New_York', 'Europe/London'). Defaults to "Asia/Shanghai" if not provided.
- **Response Structure**: Returns the current time in the format "YYYY-MM-DD HH:MM:SS" for the specified timezone.
- **Notes**: The service can be integrated with IP location services to automatically determine the user's timezone based on their IP address, providing a more personalized experience without requiring the user to manually select their timezone.
