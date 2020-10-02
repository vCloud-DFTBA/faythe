# Authentication and Authorization

- [Authentication and Authorization](#authentication-and-authorization)
  - [1. Design](#1-design)
    - [1.1. Authentication](#11-authentication)
    - [1.2. Authorization](#12-authorization)
  - [2. APIs](#2-apis)
    - [2.1. Get token](#21-get-token)
    - [2.2. Create user](#22-create-user)
    - [2.3. Delete user](#23-delete-user)
    - [2.4. List users with policies](#24-list-users-with-policies)
    - [2.5. Change user password](#25-change-user-password)
    - [2.5. Add policies](#25-add-policies)
    - [2.6. Remove policies](#26-remove-policies)

## 1. Design

Check the [proposal](./proposal/authn-authz.md).

### 1.1. Authentication

- Get Token flow:

  - User provide an username and password with HTTP Basic Authentication to issue a token.
  - In Faythe side, the Basic Authentication passwords are stored as hashed as the [bcrypt](https://en.wikipedia.org/wiki/Bcrypt) algorithm. It is your responsibility to pick the number of rounds that matches your security standards. More rounds make brute-froce more complicated at the cost of more CPU power and more time to authenticate the requests.
  - Password is checked against the hashed password stored in Etcd.
  - If a match is found, a token is created and sent back to the client in the response header with a key of "Authorization" (by default). Here is the response header format: `Authorization: Bearer <token>`.

- Restricted API request flow:

  - Client adds the Authorization token to the request header with the key "Authorization".
  - The token is checked and parsed to get the user data stored in.
  - If the token is valid, the user data will be passed in the request token context. The next middleware(s)/handler(s) can use it later.
  - If this is invalid/expired token, return an error to client.

### 1.2. Authorization

- By using [Casbin](https://casbin.org/en/), we create an authorization mechanism.
- Flow:
  - After pass the Authenticator middleware, the request jumps in Authorizor middleware.
  - Faythe retrieves the user data from the request context and get the username.
  - Using Casbin Enforcer, Faythe checks whether the authenticated user is allowed to perform the request.
  - Return a message to client if they don't have permission.
  - If OK, move to the handler.

## 2. APIs

### 2.1. Get token

**PATH**: `/public/tokens`

**METHOD**: `POST`

With Basic Auth.

If you configure the option `jwt.is_bearer_token` as `True`, the response format will be like this:

```
Authorization: Bearer <token>
```

### 2.2. Create user

**PATH**: `/users`

**METHOD**: `POST`

| Parameter | In    | Type   | Required | Default | Description     |
| --------- | ----- | ------ | -------- | ------- | --------------- |
| username  | query | string | true     |         | User's name     |
| password  | query | string | true     |         | User's password |

### 2.3. Delete user

**PATH**: `/users/{user}`

**METHOD**: `DELETE`

| Parameter | In   | Type   | Required | Default | Description |
| --------- | ---- | ------ | -------- | ------- | ----------- |
| user      | path | string | true     |         | User's name |

### 2.4. List users with policies

**PATH**: `/users`

**METHOD**: `GET`

| Parameter | In    | Type   | Required | Default | Description |
| --------- | ----- | ------ | -------- | ------- | ----------- |
| name      | query | string | true     |         | User's name |

### 2.5. Change user password

**PATH**: `/users/{user}/change_password`
**METHOD**: `PUT`

| Parameter   | In    | Type   | Required | Default | Description         |
| ----------- | ----- | ------ | -------- | ------- | ------------------- |
| user        | path  | string | true     |         | User's name         |
| newpassword | query | string | true     |         | User's new password |

### 2.5. Add policies

**PATH**: `/policies`
**METHOD**: `POST`

| Parameter     | In   | Type   | Required | Default | Description                                                                                                                                                         |
| ------------- | ---- | ------ | -------- | ------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| user          | path | string | true     |         | User's name                                                                                                                                                         |
| policies      | body | object | true     |         | A list of Policy instance                                                                                                                                           |
| policy        | body | object | true     |         | A policy instance                                                                                                                                                   |
| policy.method | body | string | true     |         | Allowed RESTful methods, can be a regex pattern. For example: `(GET)                                                                                                | (POST)`... Please check [Casbin docs](https://casbin.org/docs/en/supported-models) for details |
| policy.path   | body | string | true     |         | Allowed URL path, can be a regex pattern. For example: `/res/*`, `/res/:id/`... Please check [Casbin docs](https://casbin.org/docs/en/supported-models) for details |

For example, the request body:

```json
[
  {
    "path": "/clouds/*",
    "method": "GET"
  },
  {
    "path": "/scalers/*",
    "method": "(GET)|(POST)"
  }
]
```

### 2.6. Remove policies

**PATH**: `/policies`
**METHOD**: `DELETE`

| Parameter     | In   | Type   | Required | Default | Description                                                                                                                                                         |
| ------------- | ---- | ------ | -------- | ------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| user          | path | string | true     |         | User's name                                                                                                                                                         |
| policies      | body | object | true     |         | A list of Policy instance                                                                                                                                           |
| policy        | body | object | true     |         | A policy instance                                                                                                                                                   |
| policy.method | body | string | true     |         | Allowed RESTful methods, can be a regex pattern. For example: `(GET)                                                                                                | (POST)`... Please check [Casbin docs](https://casbin.org/docs/en/supported-models) for details |
| policy.path   | body | string | true     |         | Allowed URL path, can be a regex pattern. For example: `/res/*`, `/res/:id/`... Please check [Casbin docs](https://casbin.org/docs/en/supported-models) for details |

For example, the request body:

```json
[
  {
    "path": "/clouds/*",
    "method": "GET"
  },
  {
    "path": "/scalers/*",
    "method": "(GET)|(POST)"
  }
]
```
