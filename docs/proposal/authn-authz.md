# Faythe Implement Multiple Users Authentication & Authorization

- [Faythe Implement Multiple Users Authentication & Authorization](#faythe-implement-multiple-users-authentication--authorization)
  - [1. Summary](#1-summary)
  - [2. Design](#2-design)
    - [2.1. Multiple users](#21-multiple-users)
    - [2.2. Authentication](#22-authentication)
    - [2.3. Authorization](#23-authorization)

## 1. Summary

Issues:

- https://github.com/vCloud-DFTBA/faythe/issues/94
- https://github.com/vCloud-DFTBA/faythe/issues/90

By now, Faythe is a single-user system - admin. Admin user can do everything, assume that two users both use Faythe. Each one can be able to delete the scaler which created by another.

We should implement the mutiple users authentication & authorization mechanism.

## 2. Design

### 2.1. Multiple users

- Faythe already uses [JWT](https://jwt.io), just need to extend the current logic to handle multiple users.
- Create a new user key path `/users`.
- An User object contains:
  - User name.
  - Hashed password.
  - Id.

```go
type User struct {
  Username string `json:"username"`
  Password string `json:"password"`
  ID       string `json:"id,omitempty"`
}
```

- Faythe has its own root/admin user by configuring the config file. The admin user is allowed to perform any requests without any restriction.

### 2.2. Authentication

- Issue Token flow:

  - User provide an username and password with HTTP Basic Authentication to issue a token.
  - In Faythe side, the Basic Authentication passwords are stored as hashed as the [bcrypt](https://en.wikipedia.org/wiki/Bcrypt) algorithm. It is your responsibility to pick the number of rounds that matches your security standards. More rounds make brute-froce more complicated at the cost of more CPU power and more time to authenticate the requests.
  - Password is checked against the hashed password stored in Etcd.
  - If a match is found, a token is created and sent back to the client in the response header with a key of "Authorization" (by default). Here is the response header format: `Authorization: Bearer <token>`.

- Restricted API request:
  - Client adds the Authorization token to the request header with the key "Authorization".
  - The token is checked and parsed to get the user data stored in.
  - If the token is valid, the user data will be passed in the request token context. The next middleware(s)/handler(s) can use it later.
  - If this is invalid/expired token, return an error to client.

### 2.3. Authorization

- By using [Casbin](https://casbin.org/en/), we create an authorization mechanism.
- The Casbin model - RESTful (key match):

```ini
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = r.sub == p.sub && keyMatch(r.obj, p.obj) && regexMatch(r.act, p.act)
```

- Admin user has a policy which granted at the beginning.

```csv
p, <admin-user-name>, /*, (GET)|(POST)|(PUT)|(DELETE)
```

- As same as other components, Faythe stores the policies in Etcd using [etcd adapter](https://github.com/ntk148v/etcd-adapter). By default, Casbin stores everything in a local file, it is suitable in cluster case.
- Faythe checks whether the authenticated user is allowed to perform the request in Authorizer middleware.
- Flow:
  - After pass the Authenticator middleware, the request jumps in Authorizor middleware.
  - Faythe retrieves the user data from the request context and get the username.
  - Using Casbin Enforcer, Faythe checks whether the authenticated user is allowed to perform the request.
  - Return a message to client if they don't have permission.
  - If OK, move to the handler.
