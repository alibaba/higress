---
title: Frontend Gray
keywords: [higress, frontend gray]
description: Frontend gray plugin configuration reference
---
## Function Description
The `frontend-gray` plugin implements the functionality of user gray release on the frontend. Through this plugin, it can be used for business `A/B testing`, while the `gradual release` combined with `monitorable` and `rollback` strategies ensures the stability of system release operations.

## Runtime Attributes
Plugin execution phase: `Authentication Phase`  
Plugin execution priority: `450`

## Configuration Fields
| Name             | Data Type         | Requirements  | Default Value | Description                                                                                                 |
|-----------------|-------------------|---------------|---------------|-------------------------------------------------------------------------------------------------------------|
| `grayKey`       | string            | Optional      | -             | The unique identifier of the user ID, which can be from Cookie or Header, such as userid. If not provided, uses `rules[].grayTagKey` and `rules[].grayTagValue` to filter gray release rules. |
| `graySubKey`    | string            | Optional      | -             | User identity information may be output in JSON format, for example: `userInfo:{ userCode:"001" }`, in the current example, `graySubKey` is `userCode`. |
| `rules`         | array of object    | Required      | -             | User-defined different gray release rules, adapted to different gray release scenarios.                      |
| `rewrite`       | object            | Required      | -             | Rewrite configuration, generally used for OSS/CDN frontend deployment rewrite configurations.                |
| `baseDeployment`| object            | Optional      | -             | Configuration of the Base baseline rules.                                                                    |
| `grayDeployments` | array of object   | Optional      | -             | Configuration of the effective rules for gray release, as well as the effective versions.                     |

`rules` field configuration description:
| Name             | Data Type         | Requirements  | Default Value | Description                                                                                |
|------------------|-------------------|---------------|---------------|--------------------------------------------------------------------------------------------|
| `name`           | string            | Required      | -             | Unique identifier for the rule name, associated with `deploy.gray[].name` for effectiveness. |
| `grayKeyValue`   | array of string   | Optional      | -             | Whitelist of user IDs.                                                                    |
| `grayTagKey`     | string            | Optional      | -             | Label key for user classification tagging, derived from Cookie.                               |
| `grayTagValue`   | array of string   | Optional      | -             | Label value for user classification tagging, derived from Cookie.                             |

`rewrite` field configuration description:
> `indexRouting` homepage rewrite and `fileRouting` file rewrite essentially use prefix matching, for example, `/app1`: `/mfe/app1/{version}/index.html` represents requests with the prefix /app1 routed to `/mfe/app1/{version}/index.html` page, where `{version}` represents the version number, which will be dynamically replaced by `baseDeployment.version` or `grayDeployments[].version` during execution.  
> `{version}` will be replaced dynamically during execution by the frontend version from `baseDeployment.version` or `grayDeployments[].version`.

| Name             | Data Type         | Requirements  | Default Value | Description                           |
|------------------|-------------------|---------------|---------------|---------------------------------------|
| `host`           | string            | Optional      | -             | Host address, if OSS set to the VPC internal access address. |
| `notFoundUri`    | string            | Optional      | -             | 404 page configuration.               |
| `indexRouting`   | map of string to string | Optional  | -             | Defines the homepage rewrite routing rules. Each key represents the homepage routing path, and the value points to the redirect target file. For example, the key `/app1` corresponds to the value `/mfe/app1/{version}/index.html`. If the effective version is `0.0.1`, the access path is `/app1`, it redirects to `/mfe/app1/0.0.1/index.html`. |
| `fileRouting`    | map of string to string | Optional  | -             | Defines resource file rewrite routing rules. Each key represents the resource access path, and the value points to the redirect target file. For example, the key `/app1/` corresponds to the value `/mfe/app1/{version}`. If the effective version is `0.0.1`, the access path is `/app1/js/a.js`, it redirects to `/mfe/app1/0.0.1/js/a.js`. |

`baseDeployment` field configuration description:
| Name             | Data Type         | Requirements  | Default Value | Description                                                                                |
|------------------|-------------------|---------------|---------------|-------------------------------------------------------------------------------------------|
| `version`        | string            | Required      | -             | The version number of the Base version, as a fallback version.                           |

`grayDeployments` field configuration description:
| Name             | Data Type         | Requirements  | Default Value | Description                                                                                  |
|------------------|-------------------|---------------|---------------|----------------------------------------------------------------------------------------------|
| `version`        | string            | Required      | -             | Version number of the Gray version, if the gray rules are hit, this version will be used. If it is a non-CDN deployment, add `x-higress-tag` to the header. |
| `backendVersion` | string            | Required      | -             | Gray version for the backend, which will add `x-mse-tag` to the header of `XHR/Fetch` requests. |
| `name`           | string            | Required      | -             | Rule name associated with `rules[].name`.                                                  |
| `enabled`        | boolean           | Required      | -             | Whether to activate the current gray release rule.                                          |

## Configuration Example
### Basic Configuration
```yml
grayKey: userid
rules:
- name: inner-user
  grayKeyValue:
  - '00000001'
  - '00000005'
- name: beta-user
  grayKeyValue:
  - '00000002'
  - '00000003'
  grayTagKey: level
  grayTagValue:
  - level3
  - level5
baseDeployment:
  version: base
grayDeployments:
  - name: beta-user
    version: gray
    enabled: true
```

The unique identifier of the user in the cookie is `userid`, and the current gray release rule has configured the `beta-user` rule.  
When the following conditions are met, the version `version: gray` will be used:
- `userid` in the cookie equals `00000002` or `00000003`
- Users whose `level` in the cookie equals `level3` or `level5`  
Otherwise, use version `version: base`.

### User Information Exists in JSON
```yml
grayKey: appInfo
graySubKey: userId
rules:
- name: inner-user
  grayKeyValue:
  - '00000001'
  - '00000005'
- name: beta-user
  grayKeyValue:
  - '00000002'
  - '00000003'
  grayTagKey: level
  grayTagValue:
  - level3
  - level5
baseDeployment:
  version: base
grayDeployments:
  - name: beta-user
    version: gray
    enabled: true
```

The cookie contains JSON data for `appInfo`, which includes the field `userId` as the current unique identifier.  
The current gray release rule has configured the `beta-user` rule.  
When the following conditions are met, the version `version: gray` will be used:
- `userid` in the cookie equals `00000002` or `00000003`
- Users whose `level` in the cookie equals `level3` or `level5`  
Otherwise, use version `version: base`.

### Rewrite Configuration
> Generally used in CDN deployment scenarios.
```yml
grayKey: userid
rules:
- name: inner-user
  grayKeyValue:
  - '00000001'
  - '00000005'
- name: beta-user
  grayKeyValue:
  - '00000002'
  - '00000003'
  grayTagKey: level
  grayTagValue:
  - level3
  - level5
rewrite:
  host: frontend-gray.oss-cn-shanghai-internal.aliyuncs.com
  notFoundUri: /mfe/app1/dev/404.html
  indexRouting:
    /app1: '/mfe/app1/{version}/index.html'
    /: '/mfe/app1/{version}/index.html',
  fileRouting:
    /: '/mfe/app1/{version}'
    /app1/: '/mfe/app1/{version}'
baseDeployment:
  version: base
grayDeployments:
  - name: beta-user
    version: gray
    enabled: true
```

`{version}` will be dynamically replaced with the actual version during execution.

#### indexRouting: Homepage Route Configuration
Accessing `/app1`, `/app123`, `/app1/index.html`, `/app1/xxx`, `/xxxx` will route to '/mfe/app1/{version}/index.html'.

#### fileRouting: File Route Configuration
The following file mappings are effective:
- `/js/a.js` => `/mfe/app1/v1.0.0/js/a.js`
- `/js/template/a.js` => `/mfe/app1/v1.0.0/js/template/a.js`
- `/app1/js/a.js` => `/mfe/app1/v1.0.0/js/a.js`
- `/app1/js/template/a.js` => `/mfe/app1/v1.0.0/js/template/a.js`
