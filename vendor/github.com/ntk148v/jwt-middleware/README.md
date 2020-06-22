# JWT Middleware

- [JWT Middleware](#jwt-middleware)
  - [1. Intro](#1-intro)
  - [2. Key features](#2-key-features)
  - [3. Installation](#3-installation)
  - [4. Examples](#4-examples)
  - [5. References](#5-references)

## 1. Intro

A middleware that will check that a [JWT](https://jwt.io) is sent on the request header (the header key is customizable) and will then set the content of the JWT into the context.

This module lets you authenticate HTTP requests using JWT tokens in your Go Programming Language applications.

## 2. Key features

- Ability to **check the request header for a JWT**.
- **Decode the JWT** and set the content of it to the request context.
- **Generate the token string** for login handler.

## 3. Installation

```bash
go get github.com/ntk148v/jwt-middleware
```

## 4. Examples

You can check out working examples in the [examples folder](./examples) for the basic usage.

## 5. References

This module is strong inspired by:

- https://github.com/auth0/go-jwt-middleware
- https://github.com/adam-hanna/goLang-jwt-auth-example
- https://github.com/go-pandora/pkg/blob/master/auth/jwt/
